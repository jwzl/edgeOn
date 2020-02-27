package main

import (
	"fmt"
	"sync"
	"time"
	"errors"
	"strings"
	"k8s.io/klog"
	"github.com/jwzl/mqtt/client"
	"github.com/jwzl/wssocket/fifo"
	"github.com/jwzl/wssocket/model"
	"github.com/jwzl/edgeOn/msghub/config"	
	"github.com/jwzl/edgeOn/common"
)

const (
	//mqtt topic should has the format:
	// mqtt/dgtwin/cloud[edge]/{edgeID}/comm for communication.
	// mqtt/dgtwin/cloud[edge]/{edgeID}/control  for some control message.
	MQTT_SUBTOPIC_PREFIX	= "mqtt/dgtwin/edge"
	MQTT_PUBTOPIC_PREFIX	= "mqtt/dgtwin/cloud"
)

type Controller struct {
	EdgeID		string
	ModuleName  string
	stopChan   chan struct{}
	//mqtt client
	mqtt	*client.Client
	mutex sync.RWMutex
	/*
	* mqtt message cache 
	* store reply message with id is the parent id(=msg.Tag) of 
	* reply message.   
	*/
	msgCache	*sync.Map
	/*
	* mqtt Sync message cache 
	* store reply message with id is the parent id(=msg.Tag) of 
	* reply message.   
	*/
	SyncMsgCache	*sync.Map
	/*
	* msg ID tag.
	* Mark the message ID of the message last send.
	*/
	LastSndMsgID	string	

	// message fifo.
	messageFifo  *fifo.MessageFifo
}

func NewController(module string) *Controller {
	var messageCache sync.Map
	var syncmessageCache sync.Map

	conf := &config.MqttConfig{
		URL:	"tcp://172.21.73.155:1883",
		ClientID:		"cloud-001",
		User:			"jinxin",
		Passwd:			"jinxin",
		QOS:				2,
		Retain:			   	false,	
		MessageCacheDepth:  100,
	}
	
	c := client.NewClient(conf.URL, conf.User, conf.Passwd, conf.ClientID)
	if c == nil {
		return nil
	} 
	
	if conf.KeepAliveInterval > 0 {
		c.SetkeepAliveInterval(time.Duration(conf.KeepAliveInterval) * time.Second)
	}
	if conf.PingTimeout	 > 0 {
		c.SetPingTimeout(time.Duration(conf.PingTimeout) * time.Second)
	}
	if conf.QOS >= 0 &&  conf.QOS <= 2 {
		c.SetQOS(byte(conf.QOS))
	}
	c.SetRetain(conf.Retain)
	if conf.MessageCacheDepth > 0 {
		c.SetMessageCacheDepth(conf.MessageCacheDepth) 
	}

	tlsConfig, err := client.CreateTLSConfig(conf.CertFilePath, conf.KeyFilePath)
	if err != nil {
		klog.Infof("TLSConfig Disabled")
	}
	c.SetTlsConfig(tlsConfig)

	return &Controller{
		ModuleName:  module, 
		mqtt:		c,
		messageFifo: fifo.NewMessageFifo(0),
		stopChan:   make(chan struct{}),
		msgCache:   &messageCache,
		SyncMsgCache: &syncmessageCache,
	}	
}

func (hc * Controller)Start(){
	klog.Infof("Start the hub....")	
	
	//start the mqtt
restart_mqtt:
	err := hc.mqtt.Start() 
	if err != nil {
		klog.Warningf("Connect mqtt broker failed, retry....")
		time.Sleep(3 * time.Second)
		goto restart_mqtt
	}
	
	//Subscribe this topic.
	subTopic := fmt.Sprintf("%s/#", MQTT_SUBTOPIC_PREFIX)
	err = hc.mqtt.Subscribe(subTopic, hc.messageArrived)
	if err != nil {
		klog.Fatalf("Subscribe topic(%s) err (%v)",subTopic, err)
		return 
	}
	
	stop := make(chan struct{}, 2)

	go  hc.routeFromMqtt(stop)
	go 	hc.doprocess(stop)

	<-stop
	if hc.mqtt != nil {
		hc.mqtt.Close()
	}
} 

func (hc * Controller) Close(){
	hc.mqtt.Close()
}

func (hc * Controller) messageArrived(topic string, msg *model.Message){
	if msg == nil {
		return
	}
	klog.Infof("message Arrived. topic =(%v),  msg %v", topic, msg)
	splitString := strings.Split(topic, "/")
	if len(splitString) != 5 {
		klog.Infof("topic =(%v),  msg ignored", splitString)
		return
	} 
	if strings.Compare(splitString[4], "comm") == 0 {
		// put the model message into fifo.
		hc.messageFifo.Write(msg)
	}else{
		//TODO:
	}
}
//ReadMessage read the message from fifo. 
func (hc * Controller) ReadMessage() (*model.Message, error){
	return hc.messageFifo.Read()
}
//WriteMessage publish the message to cloud.
func (hc * Controller) WriteMessage(clientID string, msg *model.Message) error {
	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	pubTopic := fmt.Sprintf("%s/%s/comm", MQTT_PUBTOPIC_PREFIX, clientID)
	return hc.mqtt.Publish(pubTopic, msg)
}

func (hc * Controller) routeFromMqtt(stop chan struct{}){
	for {
		msg, err := hc.ReadMessage()
		if err != nil {
			klog.Errorf("failed to receive message from mqtt channel")
			stop <- struct{}{}
			break
		}

		if msg == nil {
			//msg == nil, Ignored. 		
			continue
		}

		parentID := msg.GetTag()
		if parentID == "" {
			continue
		} 	

		if msg.GetOperation() != common.DGTWINS_OPS_RESPONSE {
			if msg.GetOperation() == common.DGTWINS_OPS_SYNC {
				v, exist := hc.SyncMsgCache.Load(parentID)
				if exist {
					ch, isThisType := v.(chan *model.Message)
					if isThisType {
						//Send sync message into channel.
						ch <- msg
					}
				}
			}	
			continue
		}	

		//Store the reply message...
		hc.msgCache.Store(parentID, msg)	
	}
}

func (hc * Controller) WaitForMsgReply(msg *model.Message, timeout time.Duration)(*common.TwinResponse, error){
	timeStamp := msg.GetTimestamp()/1e3
	msgID := msg.GetID()

	now	:= time.Now().UnixNano() / 1e9

	for {
		if time.Duration(now) - time.Duration(timeStamp) >= timeout {
			return nil, errors.New("timeout")	
		}else{
			v, exist := hc.msgCache.Load(msgID)
			if exist {
				msg, isMsgType:= v.(*model.Message)
				if !isMsgType {
					return nil, errors.New("error message type")
				}

				rspMsg, err := common.UnMarshalResponseMessage(msg)
				if err != nil {
					return nil, err
				}
				// delete the msg in cache 
				hc.msgCache.Delete(msgID)
				return rspMsg, nil
			}
		}
	}
}

func (hc * Controller) doTwinMsgSend(from, to, operation, resource string, 
			twins []common.DeviceTwin, timeout time.Duration )(*common.TwinResponse, error) {
	//build twin message
	msgContent, err := common.BuildTwinMessage(twins)
	if err != nil {
		return nil, err
	}
	klog.Infof("msg content (%v)", string(msgContent))	
	modelMsg := common.BuildModelMessage(from, to, 
						operation, resource, msgContent)
	
	hc.LastSndMsgID = modelMsg.GetID()
	//Send message over mqtt.
	err = hc.WriteMessage("edge-001", modelMsg)
	if err != nil {
		return nil, err
	}
	klog.Infof("Send successful")
	//wait for reply
	rspMsg, err:= hc.WaitForMsgReply(modelMsg, timeout)
	if err != nil {
		return nil, err
	}
	
	return rspMsg, nil			
}
/*
* updateDevice:
* create twin just only need the twin ID.  
*/
func (hc * Controller) CreateTwins(iscloud bool, twinID string, 
							timeout time.Duration) (*common.TwinResponse, error) {
	from := hc.ModuleName

	twins := []common.DeviceTwin{common.DeviceTwin{ID: twinID}}	
	return hc.doTwinMsgSend(from, common.TwinModuleName, 
				common.DGTWINS_OPS_CREATE, common.DGTWINS_RESOURCE_TWINS, twins, timeout)		
}

/*
* Delete twin.
*/
func (hc * Controller) DeleteTwins(iscloud bool, twinID	string, 
					timeout time.Duration) (*common.TwinResponse, error) {
	from := hc.ModuleName
	
	twins := []common.DeviceTwin{common.DeviceTwin{ID: twinID}}
	return hc.doTwinMsgSend(from, common.TwinModuleName, 
				common.DGTWINS_OPS_DELETE, common.DGTWINS_RESOURCE_TWINS, twins, timeout)
}

/*
* Get all twin information.
*/
func (hc * Controller) GetDevice(iscloud bool, twinID []string, 
				timeout time.Duration) (*common.TwinResponse, error){

	twins := make([]common.DeviceTwin, 0)
	from := hc.ModuleName

	for _, id := range twinID {
		twins = append(twins,  common.DeviceTwin{
			ID: id,	
		})
	}

	return hc.doTwinMsgSend(from, common.TwinModuleName, 
				common.DGTWINS_OPS_GET, common.DGTWINS_RESOURCE_TWINS, twins, timeout)
}

/*
* update twin's property.
*/
func (hc * Controller) UpdateProperty(deviceID string, properties  []common.TwinProperty,
									 timeout time.Duration) (*common.TwinResponse, error){

	from := hc.ModuleName
	twin := common.DeviceTwin{
		ID: deviceID,
		Properties: common.DeviceTwinProperties{
			Desired: properties,
		},
	}	
 	twins := []common.DeviceTwin{twin}

	return hc.doTwinMsgSend(from, common.TwinModuleName, 
				common.DGTWINS_OPS_UPDATE, common.DGTWINS_RESOURCE_PROPERTY, twins, timeout)	
}

/*
* Delete twin's property.
*/
func (hc * Controller) DeleteProperty(iscloud bool, twin *common.DeviceTwin, 
							timeout time.Duration)(*common.TwinResponse, error){

	var from string
	twins := []common.DeviceTwin{*twin}
 
	if iscloud {
		from  = common.CloudName 
	}else{ 
		from  = common.EdgeAppName
	}

	return hc.doTwinMsgSend(from, common.TwinModuleName, 
				common.DGTWINS_OPS_DELETE, common.DGTWINS_RESOURCE_PROPERTY, twins, timeout)
}

/*
* Get twin's property.
*/
func (hc * Controller) GetProperty(iscloud bool, twin *common.DeviceTwin, 
							timeout time.Duration)(*common.TwinResponse, error){

	var from string
	twins := []common.DeviceTwin{*twin}
 
	if iscloud {
		from  = common.CloudName 
	}else{ 
		from  = common.EdgeAppName
	}

	return hc.doTwinMsgSend(from, common.TwinModuleName, 
				common.DGTWINS_OPS_GET, common.DGTWINS_RESOURCE_PROPERTY, twins, timeout)
}

/*
* Watch twin's property.
* Watch's call sequence as below:
* 	parentID, .. := WatchProperty(.....)
*
* 	for {
*		rsp, err := GetSyncResult(parentID, timeout)
*		# process the rsp message....	
*   }
*	
*	## stop the watch.
*/
/*func (hc * Controller) WatchProperty(iscloud bool, twins []*types.DigitalTwin, 
							timeout time.Duration)(*types.DGTwinResponse, error, string){
	
	var from string
 
	if iscloud {
		from  = types.CloudName 
	}else{ 
		from  = types.EdgeAppName
	}

	rsp, err := hc.doTwinMsgSend(from, types.TwinModuleName, 
				types.DGTWINS_OPS_WATCH, types.DGTWINS_RESOURCE_PROPERTY, twins, timeout)
	parentID := hc.LastSndMsgID
	if err != nil {
		//Create sync channel
		hc.createSyncChannel(parentID)
	}	

	return rsp, err, parentID
}

func (hc * Controller) createSyncChannel(parentID string){
	_ , exist := hc.SyncMsgCache.Load(parentID)
	if exist {
		hc.SyncMsgCache.Delete(parentID)
	}

	ch := make(chan *model.Message, 128)
	hc.SyncMsgCache.Store(parentID, ch)
}*/
/*
* Get the watch's result.
*/
/*
func (hc * Controller) GetSyncResult(parentID string, timeout time.Duration)(*types.DGTwinResponse, error){
	v , exist := hc.SyncMsgCache.Load(parentID)
	if !exist {
		return nil, errors.New("No such Channel.")
	}
	ch, isThisType := v.(chan *model.Message)
	if !isThisType {
		return nil, errors.New("No Channel Type.")
	}

	select {
	case msg, ok := <-ch:
		if !ok {
			return nil, errors.New("Channel is closed.")
		}
		
		// UnMarshal the response message.
		rspMsg, err := types.UnMarshalResponseMessage(msg)
		if err != nil {
			return nil, err
		}
		
		return rspMsg, nil
	case <-time.After(timeout):
		klog.Warningf("wait timeout!") 
	}

	return nil, errors.New("time out.")
} 
*/

/*
* TODO: stop Watch.
*/

func (hc * Controller) doprocess(stop chan struct{}) error{

loop:
	//Create the device. 
	rsp, err := hc.CreateTwins(true, "00-0B-AB-C0-C2-C4", 2*time.Second)
	if err != nil {
		klog.Warningf("create device failed with err (%v)", err)
		return err
	}

	if rsp.Code != common.RequestSuccessCode {
		klog.Warningf("create device failed with err (%d, %s)", rsp.Code, rsp.Reason)
		return err
	}

	klog.Infof("create device Successful (%v)", rsp)
	
	time.Sleep(3 * time.Second)
	/* 
	* Get the device.
	* After create the device, edgetwin module will detect  device and match the real physical
	* device, and then, physical device will report all information to  edgetwin module, So, we 
	* will wait these operation is done, then getDevice will get all information about the device.   
	*/
	twinIDs := []string{"00-0B-AB-C0-C2-C4"}
	rsp, err = hc.GetDevice(true, twinIDs, 2*time.Second)
	if err != nil {
		klog.Warningf("Get device failed with err (%v)", err)
		return err
	}
	
	klog.Infof("######## Get the result")
	dev := &rsp.Twins[0]
	klog.Infof("##### Device  Information: ", dev)	
	desiredProps := common.GetDesiredProperties(dev)
	//reportedProps := common.GetReportedProperties(dev)

	prop := common.GetPropertyValue(desiredProps, "led_pin0")
	prop.Value[0] = 0x50
	prop.Value[1] = 0xA5	

	//update the property
	properties:= []common.TwinProperty{*prop}
	rsp, err = hc.UpdateProperty("00-0B-AB-C0-C2-C4", properties, 2*time.Second)
	if err != nil {
		klog.Warningf("UpdateProperty failed with err (%v)", err)
		return err
	}
	if rsp.Code != common.RequestSuccessCode {
		klog.Warningf("create device failed with err (%d, %s)", rsp.Code, rsp.Reason)
		return err
	}

	// Get the property;

	// Watch the propeerty.

	// delete the porperty  


	time.Sleep(3 * time.Second)
	rsp, err = hc.DeleteTwins(true, "00-0B-AB-C0-C2-C4", 2*time.Second)
	if err != nil {
		klog.Warningf("Delete device failed with err (%v)", err)
		return err
	}
	if rsp.Code != common.RequestSuccessCode {
		klog.Warningf("delete device failed with err (%d, %s)", rsp.Code, rsp.Reason)
		return err
	}
	klog.Infof("delete device Successful (%v)", rsp)
	
	rsp, err = hc.GetDevice(true, twinIDs, 2*time.Second)
	if err != nil {
		klog.Warningf("Get device failed with err (%v)", err)
		return err
	}
	
	time.Sleep(3 * time.Second)
	goto loop

	return nil
}
/*
* main
*/
func main() {
	controller := NewController(common.CloudName)

	//Start the cloud test over mqtt.
	controller.Start() 
}

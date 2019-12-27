/*
Copyright 2019 The edgeOn Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package dtcontroller

import (
	"reflect"	
	"testing"
	"github.com/jwzl/beehive/pkg/core/context"
	"github.com/jwzl/edgeOn/dgtwin/types"
	"github.com/jwzl/edgeOn/dgtwin/dtcontext"
)

func TestNewDGTwinController(t *testing.T){
	c := context.GetContext(context.MsgCtxTypeChannel)
	ctx := dtcontext.NewDTContext(c)
	if ctx == nil {
		t.Errorf("create dtcontext error")
		return
	}

	tests := []struct {
		name	string
		want	*DGTwinController
		list 	[]string
	}{
		{
			name:	"NewDGTwinController",
			want:	&DGTwinController{
				ID: "",
				Stop: make(chan bool, 1),
				context: ctx,
			},
			list:	[]string {types.DGTWINS_MODULE_COMM, types.DGTWINS_MODULE_TWINS, types.DGTWINS_MODULE_PROPERTY},				
		},
	}

	for _ , test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := NewDGTwinController("", c) 
			if got == nil {
				t.Errorf("create controller == nil")
				return
			}

			if reflect.TypeOf(got) != reflect.TypeOf(test.want){
				t.Errorf("NewDGTwinController() =%v, want = %v",got, test.want)
				return
			}

			if !reflect.DeepEqual(got.ID, test.want.ID) {
				t.Errorf("got.ID = %v, test.want.ID= %v",got.ID, test.want.ID)
				return
			}

			if got.context == nil {
				t.Errorf("got.context ==nil ")
				return
			}
			
			for _, module := range test.list {
				if _, exist:= got.context.Modules[module]; !exist {
					t.Errorf("module %s is missing ",module)
					return
				}
			} 

			if cap(got.Stop) != cap(test.want.Stop) {
				t.Errorf("failed due to wrong Stop Chan Size,Got = %v Want =%v", got.Stop, test.want.Stop)
			}
		})
	}
}



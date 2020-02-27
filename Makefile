.PHONY:	edgeOn

edgeOn:
	@export GO111MODULE=on && \
	export GOPROXY=https://goproxy.io && \
	go build edgeOn.go
	@chmod 777 edgeOn


.PHONY: clean
clean:
	@rm -rf edgeOn
	@echo "[clean Done]"

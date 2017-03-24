heroku:
	go get -u github.com/FiloSottile/gvt
	cd $(GOPATH)/src/github.com/travis-ci/lb-monitor
	gvt restore
	go build -o lb-monitor
	cd -
	cp $(GOPATH)/src/github.com/travis-ci/lb-monitor/lb-monitor .

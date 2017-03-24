heroku:
	go get -u github.com/FiloSottile/gvt
	go get -u github.com/travis-ci/lb-monitor
	cd $(GOPATH)/src/github.com/travis-ci/lb-monitor
	gvt restore
	go build -o lb-monitor
	ls -lah
	ls -lah $(GOPATH)/src/github.com/travis-ci/lb-monitor
	cd -
	cp $(GOPATH)/src/github.com/travis-ci/lb-monitor/lb-monitor .

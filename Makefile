heroku:
	make -C $(GOPATH)/src/github.com/travis-ci/lb-monitor
	cp $(GOPATH)/src/github.com/travis-ci/lb-monitor/lb-monitor .

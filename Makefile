heroku:
	go get -u github.com/FiloSottile/gvt
	pushd $(GOPATH)/src/github.com/travis-ci/lb-monitor
	gvt restore
	go build -o lb-monitor
	popd
	cp $(GOPATH)/src/github.com/travis-ci/lb-monitor/lb-monitor .

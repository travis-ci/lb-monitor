heroku:
	go get -u github.com/travis-ci/lb-monitor
	go build -o lb-monitor github.com/travis-ci/lb-monitor

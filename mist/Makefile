all: templates

templates: public/js/templates.js static.go

static.go: public/* public/js/*
	go generate

public/js/templates.js: templates/client/*
	(cd templates/client && hulk *.html > ../../public/js/templates.js)
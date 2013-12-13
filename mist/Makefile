all: templates

templates: public/js/templates.js

public/js/templates.js: templates/client/*
	(cd templates/client && hulk *.html > ../../public/js/templates.js)
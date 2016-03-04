#!/bin/bash

# Add Travis badge:
cat > ./README.md << 'EOF'
[![Build Status](https://travis-ci.org/PromonLogicalis/snmp.svg?branch=master)](https://travis-ci.org/PromonLogicalis/snmp) [![Go Report Card](https://goreportcard.com/badge/github.com/PromonLogicalis/snmp)](https://goreportcard.com/report/github.com/PromonLogicalis/snmp) [![GoDoc](https://godoc.org/github.com/PromonLogicalis/snmp?status.svg)](https://godoc.org/github.com/PromonLogicalis/snmp)
EOF

# Add Go doc
godocdown ./ >> ./README.md

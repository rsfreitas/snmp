#!/bin/bash

# Add Travis badge:
cat > ./README.md << 'EOF'
[![Build Status](https://travis-ci.org/PromonLogicalis/snmp.svg?branch=master)](https://travis-ci.org/PromonLogicalis/snmp)
EOF

# Add Go doc
godocdown ./ >> ./README.md

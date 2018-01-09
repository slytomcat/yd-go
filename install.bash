#!/bin/bash

cp YD.go /usr/bin/yd
mkdir -p /usr/share/yd.go/icons/dark
mkdir -p /usr/share/yd.go/icons/light
cp icons/yd* /usr/share/yd.go/icons
cp icons/dark/* /usr/share/yd.go/icons/dark/
cp icons/light/* /usr/share/yd.go/icons/light/
cp README.md /usr/share/yd.go/


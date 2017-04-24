#!/bin/sh
sudo networksetup -setwebproxystate "Wi-Fi" on
sudo networksetup -setsecurewebproxystate "Wi-Fi" on
sudo networksetup -setwebproxy "Wi-Fi" 127.0.0.1 8888
sudo networksetup -setsecurewebproxy "Wi-Fi" 127.0.0.1 8888

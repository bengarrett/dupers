#!/usr/bin/env bash
# macOS example script for video

clear
echo ""
echo "* Immediate duplicate matching"
sleep 1
echo ""
echo dupers dupe ~/Desktop/15368.png /Volumes/PortableSSD/example/
./dupers dupe ~/Desktop/15368.png /Volumes/PortableSSD/example/
sleep 2
echo "* Instant filename searching"
sleep 1
echo ""
echo dupers -name search ".png" /Volumes/PortableSSD/example/
echo ""
./dupers -name search ".png" /Volumes/PortableSSD/example/
echo ""
sleep 2
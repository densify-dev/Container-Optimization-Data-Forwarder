#!/bin/sh
lastAppMsg="dataCollection, not forwarding data to Densify"
/home/densify/bin/dataCollection --file config --path /home/densify/config
rc=$?
if [ $rc -eq 0 ]; then
  lastAppMsg="forwarder, data collected but not forwarded to Densify"
  /home/densify/bin/forwarder
  rc=$?
fi
if [ $rc -ne 0 ]; then
  echo "Error: got return code $rc from $lastAppMsg"
fi
exit $rc

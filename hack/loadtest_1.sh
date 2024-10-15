#!/bin/bash

# if you came here to break it, you're in the right place
for i in {1..10}; do
    curl -X POST localhost:4002/api/chains -H 'content-type: application/json' --data-binary '{"userMessage": "add '$RANDOM' and '$RANDOM'"}'
    sleep .1
done


# https://www.loom.com/share/b70bad3c91c542bdb36a949424910c35

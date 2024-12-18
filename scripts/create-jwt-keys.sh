#!/bin/sh

PRIVATE_KEY_FILE=jwt-private-key.pem
PUBLIC_KEY_FILE=jwt-public-key.pem

ssh-keygen -t rsa -m pem -f $PRIVATE_KEY_FILE
rm $PRIVATE_KEY_FILE.pub
openssl rsa -in $PRIVATE_KEY_FILE -pubout -out $PUBLIC_KEY_FILE
#!/bin/sh

ssh-keygen -t rsa -m pem -f jwt-private-key.pem
rm jwt-private-key.pem.pub
openssl rsa -in jwt-private-key.pem -pubout -out jwt-public-key.pem
#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-mongodb
  make test
popd
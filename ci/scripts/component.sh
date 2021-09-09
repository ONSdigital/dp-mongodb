#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-mongodb/mongodb
  make test-component
popd
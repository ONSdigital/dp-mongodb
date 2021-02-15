#!/bin/bash -eux

cwd=$(pwd)

pushd $cwd/dp-mongodb
  make audit
popd
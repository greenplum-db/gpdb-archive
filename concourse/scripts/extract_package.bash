#!/usr/bin/env bash

extract_rpm_to_tar () {
    rpm2cpio gpdb_package/greenplum-db-*.rpm | cpio -idvm

    local tarball="${PWD}/gpdb_artifacts/bin_gpdb.tar.gz"
    pushd usr/local/greenplum-db-*
        tar czf "${tarball}" ./*
    popd
}
extract_deb_to_tar () {
  ar -x gpdb_package/greenplum-db-*.deb
  tar -xf data.tar.xz
  local tarball="${PWD}/gpdb_artifacts/bin_gpdb.tar.gz"
  pushd usr/local/greenplum-db-*
      tar czf "${tarball}" ./*
  popd
}

copy_bin_gpdb () {
  cp gpdb_package/bin_gpdb.tar.gz gpdb_artifacts/bin_gpdb.tar.gz
}

#if test -n "$(find gpdb_package -maxdepth 1 -name '*.rpm' -print -quit)"
#then
#    extract_rpm_to_tar "${@}"
#else
#    extract_deb_to_tar "${@}"
#fi

## FIXME: we copy a gpdb binary from some dev pipeline instead of extracting from
## a product release, in order to workaround the binswap test. Revert this change,
## when 7.2.0 is released and point pivotal-gpdb-pivnet-product-version to 7.2.0.
copy_bin_gpdb

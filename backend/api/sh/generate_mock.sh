#!/bin/bash -eu

ROOT="./"
MOCK_DATA_ROOT="./generated/mock/"

SEARCH_DIR_LIST=$(ls -l ${ROOT} | grep ^d | awk '{print $9}')

echo "Start..."

cd ${ROOT}

for dir in ${SEARCH_DIR_LIST}; do
  # find go source file without mock file and test file
  file_path_list=$(find ./${dir} -type f -not -name "mock_*.go" -not -name "*_test.go" -name "*.go")

  for file_path in ${file_path_list}; do
    # generate mock file
    echo $file_path
    mockgen -package mock -source=./$file_path -destination=./$MOCK_DATA_ROOT/$file_path
  done
done

echo "Done."

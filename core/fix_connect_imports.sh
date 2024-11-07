#!/bin/bash
# Find and remove trailing slashes from import paths in generated Go files

for file in $(find pb -type f -name '*.go'); do
  sed 's|pb "epistemic-me-core/pb/"|pb "epistemic-me-core/pb"|' $file > temp.go
  mv temp.go $file
done

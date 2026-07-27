#!/bin/sh
set -eu

if [ "$#" -ne 1 ]; then
  echo "usage: $0 OUTPUT_DIRECTORY" >&2
  exit 2
fi

output_directory=$1
project_directory="$output_directory/project"
module_directory="$output_directory/go-modules"

rm -rf "$output_directory"
mkdir -p "$project_directory" "$module_directory"

cp LICENSE NOTICE THIRD_PARTY_NOTICES.md "$project_directory/"

main_module=$(go list -m -f '{{.Path}}')
module_inventory=$(mktemp)
trap 'rm -f "$module_inventory"' EXIT

go list -deps -f '{{with .Module}}{{.Path}}|{{.Dir}}{{end}}' \
  ./cmd/signalforge-workspace |
  LC_ALL=C sort -u > "$module_inventory"

module_count=0
while IFS='|' read -r module module_root; do
  if [ -z "$module" ] || [ "$module" = "$main_module" ]; then
    continue
  fi

  safe_module=$(printf '%s' "$module" | sed 's/[^A-Za-z0-9._-]/_/g')
  destination="$module_directory/$safe_module"
  mkdir -p "$destination"
  found=0

  for candidate in \
    LICENSE LICENSE.txt LICENSE.md \
    COPYING COPYING.txt COPYING.md \
    NOTICE NOTICE.txt NOTICE.md
  do
    if [ -f "$module_root/$candidate" ]; then
      cp "$module_root/$candidate" "$destination/$candidate"
      found=1
    fi
  done

  if [ "$found" -ne 1 ]; then
    echo "no license or notice file found for runtime module: $module" >&2
    exit 1
  fi

  printf '%s\t%s\n' "$module" "$safe_module" >> "$output_directory/GO_MODULES.tsv"
  module_count=$((module_count + 1))
done < "$module_inventory"

if [ "$module_count" -eq 0 ]; then
  echo "runtime module inventory is empty" >&2
  exit 1
fi

LC_ALL=C sort "$output_directory/GO_MODULES.tsv" \
  -o "$output_directory/GO_MODULES.tsv"

(
  cd "$output_directory"
  find . -type f ! -name SHA256SUMS -print |
    LC_ALL=C sort |
    xargs sha256sum > SHA256SUMS
)

printf 'runtime license bundle: %s Go modules\n' "$module_count"

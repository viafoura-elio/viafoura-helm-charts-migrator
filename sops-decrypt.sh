#!/usr/bin/env bash

set -euo pipefail

export AWS_PROFILE=cicd-terraform

# Parse command line arguments
TARGET="${1:-}"

# Determine the search path
if [[ -n "${TARGET}" ]]; then
  # If target is provided, use it
  if [[ -f "${TARGET}" ]]; then
    # Target is a file
    SEARCH_PATH="${TARGET}"
  elif [[ -d "${TARGET}" ]]; then
    # Target is a directory
    SEARCH_PATH="${TARGET}"
  else
    echo "Error: Target '${TARGET}' does not exist or is not accessible." >&2
    exit 1
  fi
else
  # No target provided, use current directory
  SEARCH_PATH="$(pwd)"
fi

# Only add files to the array if they actually exist
SOPS_DEC_FILES=()
if [[ -f "${SEARCH_PATH}" && "${SEARCH_PATH}" == *.enc.yaml ]]; then
  # Single file target
  SOPS_DEC_FILES=("${SEARCH_PATH}")
elif [[ -d "${SEARCH_PATH}" ]]; then
  # Directory target - find all .enc.yaml files
  if mapfile -t FOUND_FILES < <(find "${SEARCH_PATH}" -type f -name '*.enc.yaml' -print | sort -u); then
    SOPS_DEC_FILES=("${FOUND_FILES[@]}")
  fi
else
  echo "Error: Invalid target '${SEARCH_PATH}'" >&2
  exit 1
fi

if [[ ${#SOPS_DEC_FILES[@]} -eq 0 ]]; then
  echo "No .enc.yaml files found for decryption."
  exit 0
fi

echo
# Process in parallel with a maximum of 4 processes at a time
for work_file in "${SOPS_DEC_FILES[@]}"; do
  (
    current_filename=$(basename "${work_file}")
    secret_dir=$(dirname "${work_file}")
    decrypted_filename="${current_filename/.enc./.dec.}"
    decrypted_file="${secret_dir}/${decrypted_filename}"

    echo "Decrypting: ${work_file} -> ${decrypted_file}"

    # Check if sops is available
    if ! command -v sops &> /dev/null; then
        echo "Error: sops is not installed or not in PATH" >&2
        exit 1
    fi

    # Decrypt with error handling - using the --input-type and --output-type flags
    if ! sops --input-type yaml --output-type yaml -d "${work_file}" > "${decrypted_file}"; then
        echo "Error: Failed to decrypt ${work_file}" >&2
        exit 1
    fi

  ) &

  # Limit the number of background processes
  if [[ $(jobs -r -p | wc -l) -ge 4 ]]; then
    wait -n
  fi
done

unset AWS_PROFILE

# Wait for all background processes to finish
wait
echo "All decryption jobs completed."

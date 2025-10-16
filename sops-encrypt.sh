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
if [[ -f "${SEARCH_PATH}" && "${SEARCH_PATH}" == *.dec.yaml ]]; then
  # Single file target
  SOPS_DEC_FILES=("${SEARCH_PATH}")
elif [[ -d "${SEARCH_PATH}" ]]; then
  # Directory target - find all .dec.yaml files
  if mapfile -t FOUND_FILES < <(find "${SEARCH_PATH}" -type f -name '*.dec.yaml' -print | sort -u); then
    SOPS_DEC_FILES=("${FOUND_FILES[@]}")
  fi
else
  echo "Error: Invalid target '${SEARCH_PATH}'" >&2
  exit 1
fi

if [[ ${#SOPS_DEC_FILES[@]} -eq 0 ]]; then
  echo "No .dec.yaml files found for encryption."
  exit 0
fi

echo
# Process in parallel with a maximum of 4 processes at a time
for work_file in "${SOPS_DEC_FILES[@]}"; do
  (
    current_filename=$(basename "${work_file}")
    secret_dir=$(dirname "${work_file}")
    encrypted_filename="${current_filename/.dec./.enc.}"
    encrypted_file="${secret_dir}/${encrypted_filename}"

    echo "Encrypting: ${work_file} -> ${encrypted_file}"

    # Check if sops is available
    if ! command -v sops &> /dev/null; then
        echo "Error: sops is not installed or not in PATH" >&2
        exit 1
    fi

    # First decrypt the existing encrypted file to compare content
    temp_decrypted=$(mktemp)
    content_changed=true
    
    if [[ -f "${encrypted_file}" ]]; then
        if sops --input-type yaml --output-type yaml -d "${encrypted_file}" > "${temp_decrypted}" 2>/dev/null; then
            # Compare content excluding SOPS metadata (remove lines starting with sops:)
            temp_filtered_orig=$(mktemp)
            temp_filtered_dec=$(mktemp)
            
            # Filter out sops metadata from both files
            grep -v '^\s*sops:' "${work_file}" | grep -v '^\s*kms:' | grep -v '^\s*lastmodified:' | grep -v '^\s*mac:' | grep -v '^\s*version:' | grep -v '^\s*created_at:' | grep -v '^\s*enc:' | grep -v '^\s*aws_profile:' | grep -v '^\s*arn:' | grep -v '^\s*shamir_threshold:' | grep -v '^\s*unencrypted_suffix:' | grep -v '^\s*-\s*arn:' > "${temp_filtered_orig}"
            grep -v '^\s*sops:' "${temp_decrypted}" | grep -v '^\s*kms:' | grep -v '^\s*lastmodified:' | grep -v '^\s*mac:' | grep -v '^\s*version:' | grep -v '^\s*created_at:' | grep -v '^\s*enc:' | grep -v '^\s*aws_profile:' | grep -v '^\s*arn:' | grep -v '^\s*shamir_threshold:' | grep -v '^\s*unencrypted_suffix:' | grep -v '^\s*-\s*arn:' > "${temp_filtered_dec}"
            
            if diff -q "${temp_filtered_orig}" "${temp_filtered_dec}" >/dev/null 2>&1; then
                content_changed=false
                echo "Skipping: ${work_file} (content unchanged)"
            fi
            
            rm -f "${temp_filtered_orig}" "${temp_filtered_dec}"
        fi
    fi
    
    rm -f "${temp_decrypted}"
    
    if [[ "${content_changed}" == "true" ]]; then
        # Encrypt with error handling - using the --input-type and --output-type flags
        if ! sops --input-type yaml --output-type yaml -e "${work_file}" > "${encrypted_file}"; then
            echo "Error: Failed to encrypt ${work_file}" >&2
            exit 1
        fi
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
echo "All encryption jobs completed."

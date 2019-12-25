#!/bin/bash

config_values_json_patch=()
values_json_patch=()

function values::json_patch() {
  set -f
  if [[ "$1" == "--config" ]] ; then
    shift
    config_values_json_patch+=($(jq -nec --arg op "$1" --arg path "$2" --arg value "${3:-""}" \
                                '{"op": $op, "path": $path} + if (($value | length) > 0) then {"value": (try ($value | fromjson) catch $value)} else {} end'))

    echo "${config_values_json_patch[@]}" | \
      jq -sec '.' > $CONFIG_VALUES_JSON_PATCH_PATH
  else
    values_json_patch+=($(jq -nec --arg op "$1" --arg path "$2" --arg value "${3:-""}" \
                                '{"op": $op, "path": $path} + if (($value | length) > 0) then {"value": (try ($value | fromjson) catch $value)} else {} end'))

    echo "${values_json_patch[@]}" | \
      jq -sec '.' > $VALUES_JSON_PATCH_PATH
  fi
  set +f
}

function values::get() {
  local values_path=$VALUES_PATH
  local required=no

  while true ; do
    case ${1:-} in
      --config)
        values_path=$CONFIG_VALUES_PATH
        shift
        ;;
      --required)
        required=yes
        shift
        ;;
      *)
        break
        ;;
    esac
  done

  local value=$(cat $values_path | jq ".${1:-}" -r)

  if [[ "$required" == "yes" ]] && values::is_empty "$value" ; then
    >&2 echo "Error: Value $1 required, but empty"
    return 1
  else
    echo "$value"
    return 0
  fi
}

function values::set() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  values::json_patch $config add $(values::normalize_path_for_json_patch $1) "$2"
}

function values::has() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  local splittable_path=$(echo "$1" | sed -e s/\'/\"/g -e ':loop' -e 's/"\([^".]\+\)\.\([^"]\+\)"/"\1##DOT##\2"/g' -e 't loop')

  local path=.$(echo "${splittable_path}" | rev | cut -d. -f2- | rev | sed -e 's/##DOT##/./g')
  local key=$(echo "${splittable_path}" | rev | cut -d. -f1 | rev | sed -e 's/##DOT##/./g' -e "s/[\"\']//g")

  if [[ "$(values::get $config | jq $path' | has("'$key'")' -r)" == "true" ]] ; then
    return 0
  else
    return 1
  fi
}

function values::unset() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  if values::has $config $1 ; then
    values::json_patch $config remove $(values::normalize_path_for_json_patch $1)
  fi
}

function values::is_empty() {
  [[ -z "${1:-}" || "${1:-}" == "null" ]]
}

function values::require_in_config() {
  if ! values::has --config $1 ; then
    >&2 echo "Error: $1 is required in config!"
    return 1
  fi
}

function values::array_has() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  values::get $config $1 | jq '(type == "array") and (index("'$2'") != null)' -e > /dev/null
}

function values::is_true() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  values::get $config $1 | jq '. == true' -e > /dev/null
}

function values::is_false() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  values::get $config $1 | jq '. == false' -e > /dev/null 2> /dev/null
}

function values::generate_password() {
  pwgen -s 20 1
}

function values::get_first_defined() {
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  for var in "$@"
  do
    if values::has $config $var ; then
      values::get $config $var
      return 0
    fi
  done
  return 1
}

function values::normalize_path_for_json_patch() {
  echo /$1 | sed -e s/\'/\"/g -e ':loop' -e 's/"\([^".]\+\)\.\([^"]\+\)"/"\1##DOT##\2"/g' -e 't loop' -e s/\"//g -e 's/\./\//g' -e 's/##DOT##/./g'
}

function values::store::replace_row_by_key() {
  # [--config] <path> <key> <row>
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  KEY_VALUE=$(jq -rn --argjson row_values "$3" '$row_values | .'$2 )
  if INDEX=$(values::get $config $1 | jq -er 'to_entries[] | select(.value.'$2' == "'$KEY_VALUE'") | .key'); then
    values::json_patch $config remove $(values::normalize_path_for_json_patch $1)/$INDEX
    values::json_patch $config add $(values::normalize_path_for_json_patch $1)/$INDEX "$3"
  else
    values::json_patch $config add $(values::normalize_path_for_json_patch $1)/- "$3"
  fi
}

function values::store::unset_row_by_key() {
  # [--config] <path> <key> <row>
  local config=""
  if [[ "$1" == "--config" ]] ; then
    config=$1
    shift
  fi

  KEY_VALUE=$(jq -rn --argjson row_values "$3" '$row_values | .'$2 )
  if INDEX=$(values::get $config $1 | jq -er 'to_entries[] | select(.value.'$2' == "'$KEY_VALUE'") | .key'); then
    values::json_patch $config remove $(values::normalize_path_for_json_patch $1)/$INDEX
  fi
}
#!/bin/sh

set -u # Undefined variables are errors

main() {
    assert_cmds
    set_globals
    handle_command_line_args "$@"
}

set_globals() {
    #Environment sanity checks
    assert_nz "$HOME" "\$HOME is undefined"
    assert_nz "$0" "\$0 is undefined"

    #script version
    version=0.0.1

    #location of the distribution server
    dist_server="https://github.com/moncho/dry/releases/download"
    version_file_url="https://raw.githubusercontent.com/moncho/dry/master/APPVERSION"

    #Install prefix
    default_prefix="${DRY_PREFIX-/usr/local/bin}"

    #Downloads go here
    dl_dir="/tmp"

    flag_verbose=false

    if [ -n "${VERBOSE-}" ]; then
	     flag_verbose=true
    fi
}

handle_command_line_args() {
  local _prefix="$default_prefix"
  local _help=false

  for arg in "$@"; do
  	case "$arg" in
  	    --help )
  		_help=true
  		;;

  	    --verbose)
  		# verbose is a global flag
  		flag_verbose=true
  		;;

  	    --version)
  		echo "dry.sh $version"
  		exit 0
  		;;

  	esac

  	if is_value_arg "$arg" "prefix"; then
      _prefix="$(get_value_arg "$arg")"
    fi
  done
  if [ "$_help" = true ]; then
    print_help
    exit 0
  fi

  download_install_dry "$_prefix"

}

is_value_arg() {
    local _arg="$1"
    local _name="$2"

    echo "$arg" | grep -q -- "--$_name="
    return $?
}

get_value_arg() {
    local _arg="$1"

    echo "$_arg" | cut -f2 -d=
}


# Returns 0 on success, 1 on error
download_install_dry() {
    local _prefix="$1"
    local _errors=0

    # Dry version
    get_latest_dry_version
    dry_version="$RETVAL"
    #dry_version="v0.3-beta.1"
    assert_nz "$dry_version" "dry_version from repository"

    determine_binary || return 1

    local _dry_binary="$RETVAL"
    local _dry_binary_file=""

    determine_remote_dry "$_dry_binary" || return 1

    local _remote_dry_binary="$RETVAL"
    assert_nz "$_remote_dry_binary" "remote dry binary"
    verbose_say "remote dry binary location: $_remote_dry_binary"

    # Download and install dry
    say "downloading dry binary"

    download_and_check "$_remote_dry_binary" false

    if [ $? != 0 ]; then
      say "failed to download binary"
      _errors=1
    else
      local _dry_binary_file="$RETVAL"
      assert_nz "$_dry_binary_file" "dry_binary_file"
      install_dry "$_dry_binary_file" "$_prefix"
      if [ $? != 0 ]; then
        say_err "failed to install dry"
        _errors=1
      fi
    fi

    if [ "$_errors" -ne 0 ]; then
      say_err "there were errors during the installation"
      if [ -f "$_dry_binary_file" ]; then
        run rm "$_dry_binary_file"
      fi
      return 1
    fi
    say "dry binary was copied to $_prefix, now you should 'sudo chmod 755 $_prefix/dry'"
}

install_dry() {
    local _dry_file="$1"
    local _prefix="$2"

    say "Moving dry binary to its destination"

    verbose_say "moving $_dry_file to $_prefix/dry"
    run mv "$_dry_file" "$_prefix/dry"
    if [ $? != 0 ]; then
      say "Failed to move binary to $_prefix/dry"
      return 1
    fi
    verbose_say "moved dry to its destination"
    return 0
}

determine_remote_dry() {
  local _binary=$1
  verbose_say "figuring out remote binary "

  local _remote_dry="$dist_server/$dry_version/$_binary"
  verbose_say "binary is $_remote_dry"

  RETVAL="$_remote_dry"
}

determine_binary() {

    verbose_say "figuring out dry binary "
    get_architecture || return 1
    local _arch="$RETVAL"
    assert_nz "$_arch" "arch"

    local _bin="dry-$_arch"
    verbose_say "binary is $_bin"

    RETVAL="$_bin"
}

get_architecture() {

    verbose_say "detecting architecture"

    local _ostype="$(uname -s)"
    local _cputype="$(uname -m)"
    local _isarm

    verbose_say "uname -s reports: $_ostype"
    verbose_say "uname -m reports: $_cputype"

    if [ "$_ostype" = Darwin -a "$_cputype" = i386 ]; then
	# Darwin `uname -s` lies
	if sysctl hw.optional.x86_64 | grep -q ': 1'; then
	    local _cputype=x86_64
	fi
    fi

    case "$_ostype" in

	Linux)
	    local _ostype=linux
	    ;;

	FreeBSD)
	    local _ostype=freebsd
	    ;;

	Darwin)
	    local _ostype=darwin
	    ;;

	MINGW* | MSYS*)
      local _ostype=windows
      ;;

	*)
	    err "unrecognized OS type: $_ostype"
	    ;;

    esac

    case "$_cputype" in

	i386 | i486 | i686 | i786 | x86)
            local _cputype=386
            ;;

	x86_64 | x86-64 | x64 | amd64)
            local _cputype=amd64
            ;;
    aarch64 | arm64)
            local _cputype=arm64
            ;;
    arm*)
            local _cputype=arm
            ;;
	*)
               err "unknown CPU type: $CFG_CPUTYPE"

    esac

    # Detect 64-bit linux with 32-bit userland
    if [ $_ostype = unknown-linux-gnu -a $_cputype = x86_64 ]; then
	# $SHELL does not exist in standard 'sh', so probably only exists
	# if configure is running in an interactive bash shell. /usr/bin/env
	# exists *everywhere*.
	local _bin_to_probe="${SHELL-bogus_shell}"
	if [ ! -e "$_bin_to_probe" -a -e "/usr/bin/env" ]; then
	    _bin_to_probe="/usr/bin/env"
	fi
	if [ -e "$_bin_to_probe" ]; then
	    file -L "$_bin_to_probe" | grep -q "x86[_-]64"
	    if [ $? != 0 ]; then
		local _cputype=386
	    fi
	fi
    fi

    local _arch="$_ostype-$_cputype"
    verbose_say "architecture is $_arch"

    RETVAL="$_arch"
}

get_latest_dry_version() {
  verbose_say "getting latest dry version from $version_file_url"
  RETVAL="v$(curl $version_file_url)"
}

# Downloads a remote file, returns 0 on success.
# Returns the path to the downloaded file in RETVAL.
download_and_check() {
    local _remote_name="$1"
    local _quiet="$2"

    download_file "$_remote_name" "$dl_dir" "$_quiet"

    if [ $? != 0 ]; then
        return 1
    fi

    #TODO Check download
    local _download_file="$RETVAL"

    assert_nz "$_download_file" "downloaded file"
    verbose_say "downloaded dry binary location: $_download_file"

    RETVAL="$_download_file"
}


download_file() {
    local _remote_name="$1"
    local _local_dirname="$2"
    local _quiet="$3"

    local _remote_basename="$(basename "$_remote_name")"
    assert_nz "$_remote_basename" "remote basename"

    local _local_name="$_local_dirname/$_remote_basename"

    verbose_say "downloading '$_remote_name' to '$_local_name'"
    # Invoke curl in a way that will resume if necessary
    if [ "$_quiet" = false ]; then
	     (run cd "$_local_dirname" && run curl -# -L -C - -f -O "$_remote_name")
    else
	     (run cd "$_local_dirname" && run curl -s -L -C - -f -O "$_remote_name")
    fi
    if [ $? != 0 ]; then
	     say_err "couldn't download '$_remote_name'"
	      return 1
    fi
    RETVAL="$_local_name"
}

# Help
print_help() {
echo '
Usage: dryup.sh [--verbose]

Options:

     --prefix=<path>                   Install to a specific location (default /usr/local/bin)
'
}

# Standard utilities

say() {
    echo "dryup: $1"
}

say_err() {
    say "$1" >&2
}

verbose_say() {
    if [ "$flag_verbose" = true ]; then
	     say "$1"
    fi
}

err() {
    say "$1" >&2
    exit 1
}

need_cmd() {
    if ! command -v "$1" > /dev/null 2>&1
    then err "need '$1' (command not found)"
    fi
}

need_ok() {
    if [ $? != 0 ]; then err "$1"; fi
}

assert_nz() {
    if [ -z "$1" ]; then err "assert_nz $2"; fi
}

# Run a command that should never fail. If the command fails execution
# will immediately terminate with an error showing the failing
# command.
ensure() {
    "$@"
    need_ok "command failed: $*"
}

# This is just for indicating that commands' results are being
# intentionally ignored. Usually, because it's being executed
# as part of error handling.
ignore() {
    run "$@"
}

# Runs a command and prints it to stderr if it fails.
run() {
    "$@"
    local _retval=$?
    if [ $_retval != 0 ]; then
	say_err "command failed: $*"
    fi
    return $_retval
}

# Prints the absolute path of a directory to stdout
abs_path() {
    local _path="$1"
    # Unset CDPATH because it causes havok: it makes the destination unpredictable
    # and triggers 'cd' to print the path to stdout. Route `cd`'s output to /dev/null
    # for good measure.
    (unset CDPATH && cd "$_path" > /dev/null && pwd)
}

assert_cmds() {
    need_cmd dirname
    need_cmd basename
    need_cmd mkdir
    need_cmd cat
    need_cmd curl
    need_cmd mktemp
    need_cmd rm
    need_cmd egrep
    need_cmd grep
    need_cmd file
    need_cmd uname
    need_cmd tar
    need_cmd sed
    need_cmd sh
    need_cmd mv
    need_cmd cut
    need_cmd sort
    need_cmd date
    need_cmd head
    need_cmd printf
    need_cmd touch
    need_cmd id
}

main "$@"

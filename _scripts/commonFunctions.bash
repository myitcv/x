if [ "${LOADED_COMMON_FUNCTIONS:-}" == "true" ]
then
	return
fi

autostash_or_export()
{
	if [ "$(type -t autostash || true)" == "function" ]
	then
		autostash "$@"
	else
		export "$@"
	fi
}
export -f autostash_or_export

running_on_ci_server()
{
	local res
	if [ "${TRAVIS:-}" == "true" ]
	then
		res=yes
	else
		res=no
	fi
	echo $res
}
export -f running_on_ci_server

only_run_on_ci_server()
{
	if [ $(running_on_ci_server) != "yes" ]
	then
		echo "This script can ONLY be run on the CI server"
		exit 1
	fi
}
export -f only_run_on_ci_server

LOADED_COMMON_FUNCTIONS=true

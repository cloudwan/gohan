#!/bin/bash

_clientCommands="list show create set delete"
_verbosity="0 1 2"
_outputFormat="json table"
_gohanCommandPos=1
__un_namedCommandWords="show set delete"
replaceTilde()
{
    _gohan=${COMP_WORDS[0]}
    _replacewhat="~"
    _replacewith=$HOME
    _gohan="${_gohan//$_replacewhat/$_replacewith}"
}
getUn_namedProperties()
{
    replaceTilde
    _schema_id=$1
    _un_named="$(${COMP_WORDS[0]} client "$_schema_id" list |
    grep -v -e '+-\++' -e '^[[:space:]]*$' |
    awk -v col='ID' -F '|' 'NR==1 { for (i=1; i<=NF; i++) if ($i ~ " *" col " *") { c=i; break } } NR>1 { print $c }')"
}

getProperties()
{
    replaceTilde
    _named="--output-format --verbosity --fields"
    _fields=""
    _schema_id=$1
    OIFS=$IFS
    IFS=$'\n' arr=`$_gohan client $_schema_id`
    _wasProperties=0
    for a in $arr
    do
	if [[ _wasProperties -eq 1 ]] ; then
            _curvar=`echo $a | awk '{print $2}'`
            if [[ -z "${_curvar// }" ]] ; then
		return
            else
		_curvar="$(echo -e "${_curvar}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
		_named="$_named --$_curvar"
		_fields="$_fields $_curvar"
            fi
	fi
	if [[ $a == *"Properties:"* ]] ; then
            _wasProperties=1
	elif [[ -z "${a// }" ]] ; then
            _wasProperties=0
	fi
    done
}

getAllSchemas()
{
    replaceTilde
    OIFS=$IFS
    IFS=$'\n'
    array=`$_gohan client`
    schemasArr=()
    schemas=""
    for curLine in $array
    do
	IFS='#' read -r -a arr <<< "$curLine"
	curSchema_id=`echo $arr | cut -d " " -f 3`
	schemasArr+=("$curSchema_id")
    done

    for i in "${schemasArr[@]}"
    do
	i="$(echo -e "${i}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
	schemas="$schemas $i"
    done
    IFS=$OIFS
}

gohanClientBashCompletion()
{

    c=" "
    let "c=(${COMP_CWORD}+$_gohanCommandPos) % 2"
    if [[ "${COMP_CWORD}" -lt $((_gohanCommandPos+1)) ]] ; then
	return 0
    fi
    if [[ "${COMP_WORDS[0]}" != "gohan" ]] &&  [[ "${COMP_WORDS[1]}" != "client" ]] ; then
	return 0
    fi
    replaceTilde
    _execOperation="$_gohan client";
    $_execOperation &> /dev/null;
    _exitCode=$?;
    if [[ _exitCode -ne 0 ]]; then
	return 0;
    fi
    if [[ "${COMP_CWORD}" -eq $((_gohanCommandPos+1)) ]] ; then
	getAllSchemas
	COMPREPLY=( $(compgen -W "${schemas}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -eq $((_gohanCommandPos+2)) ]] ; then
	COMPREPLY=( $(compgen -W "${_clientCommands}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ c -eq 1 ]] &&  [[ "${cur}" == -* ]] ; then
	_schema_id="${COMP_WORDS[$((_gohanCommandPos+1))]}"
	getProperties $_schema_id
	IFS=$OIFS
	COMPREPLY=( $(compgen -W "${_named}" -- ${cur} ) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ c -eq 1 ]] && [[ "${cur}" != -* ]] ; then
	toShow=0
	curCommand="${COMP_WORDS[3]}"
	[[ $__un_namedCommandWords =~ (^|[[:space:]])$curCommand($|[[:space:]]) ]] && toShow=1 || toShow=0
	if [[ $toShow == 0 ]]; then
            return 0
	fi
	device="${COMP_WORDS[$((_gohanCommandPos+1))]}"
	getUn_namedProperties $device
	IFS=$OIFS
	COMPREPLY=( $(compgen -W "${_un_named}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ c -eq 0 ]] && [[ "${prev}" == "--fields" ]] ; then
	getProperties "${COMP_WORDS[$((_gohanCommandPos+1))]}"
	IFS=$OIFS
	COMPREPLY=( $(compgen -W "${_fields}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ c -eq 0 ]] && [[ "${prev}" == "--verbosity" ]] ; then
	COMPREPLY=( $(compgen -W "${_verbosity}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ c -eq 0 ]] && [[ "${prev}" == "--output-format" ]] ; then
	COMPREPLY=( $(compgen -W "${_outputFormat}" -- ${cur}) )
	return 0

    fi
}

__gohan()
{


    declare -A possibleWord=( ["1"]="gohan"
			      ["2"]="--debug --help --version -d -h -v"
			      ["3"]="client validate v init-db idb convert conv server srv test_extensions test_ex migrate mig template run test test openapi markdown dot glace-server gsrv help h")

    declare -A commandOptions=( ["validate"]="--schema -s --json -i"
				["template"]="--config-file --split-by-resource-group --policy --template -t"
				["openapi"]="--config-file --template -t --split-by-resource-group --policy --version --title --description"
				["markdown"]="--config-file --template -t --split-by-resource-group --policy"
			      )
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    globalOptions="--debug --help --version -d -h -v"
    commands="client validate v init-db idb convert conv server srv test_extensions test_ex migrate mig template run test test openapi markdown dot glace-server gsrv help h"
    if [[ "${COMP_CWORD}" -gt 1 ]] && [[ "${COMP_WORDS[1]}" == "client" ]]; then
        gohanClientBashCompletion
        return 0;
    fi
    if [[ "${COMP_CWORD}" == 1 ]] && [[ "${cur}" == "-"* ]]; then
	curOption=2
	COMPREPLY=( $(compgen -W "${possibleWord[$curOption]}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" == 1 ]] && [[ "${cur}" != "-"* ]]; then
	curOption=3
	COMPREPLY=( $(compgen -W "${possibleWord[$curOption]}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" == 2 ]] && [[ "${prev}" == "-"* ]]; then
	curOption=3
	COMPREPLY=( $(compgen -W "${possibleWord[$curOption]}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" == 2 ]]  && [[ "${prev}" != "-"* ]]; then
	COMPREPLY=( $(compgen -W "${commandOptions[${prev}]}" -- ${cur}) )
	return 0
    elif [[ "${COMP_CWORD}" == 3 ]] && [[ "${prev}" != "-"* ]]; then
	COMPREPLY=( $(compgen -W "${commandOptions[${prev}]}" -- ${cur}) )
	return 0
    fi
}
complete -F __gohan gohan

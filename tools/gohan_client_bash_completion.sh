#!/bin/bash

_commands="list show create set delete"
_verbosity="0 1 2"
_outputFormat="json table"
_gohanCommandPos=1
__un_namedCommandWords="show set delete"
getUn_namedProperties()
{
    _schema_id=$1
    OIFS=$IFS
    IFS=$'\n'
    arr=`${COMP_WORDS[0]} client $_schema_id list`
    _wasProperties=0
    _counter=0
    _un_named=""
    for a in $arr
    do
        _cop=$a
        _replacewhat="-"
        _replacewith=""
        _replacewhat2="+"
        _result="${_cop//$_replacewhat/$_replacewith}"
        _result="${_result//$_replacewhat2/$_replacewith}"
        let _counter=_counter+1
        if [ -z "$_result" ]; then
            continue
        fi
        if [ $_counter -gt 2 ];then
            _args=$(echo $a | tr "|" "\n")
            _sub=0
            for i in $_args
            do
		let _sub=_sub+1
		if [ $_sub -eq 1 ] && [ ! -z "${i// }" ] ; then
        	    i="$(echo -e "${i}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
                    _un_named="$i $_un_named"
		fi
            done
        fi
    IFS=$OIFS
    done
}

getProperties()
{
    _named="--output-format --verbosity --fields"
    _fields=""
    _schema_id=$1
    OIFS=$IFS
    IFS=$'\n' arr=`${COMP_WORDS[0]} client $_schema_id`
    _wasProperties=0
    for a in $arr
    do
	if [[ _wasProperties -eq 1 ]] ; then
            __curvar=`echo $a | awk '{print $2}'`
            if [[ -z "${__curvar// }" ]] ; then
		return
            else
		__curvar="$(echo -e "${__curvar}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
		_named="$_named --$__curvar"
		_fields="$_fields $__curvar"
            fi
	fi
	if [[ $a == *"Properties:"* ]] ; then
            _wasProperties=1
	elif [[ -z "${a// }" ]] ; then
            _wasProperties=0
	fi
    IFS=$OIFS
    done
}

getAllSchemas()
{
    OIFS=$IFS
    IFS=$'\n'
    _array=`${COMP_WORDS[0]} client`
    _schemasArr=()
    _schemas=""
    for _curLine in $_array
    do
	IFS='#' read -r -a arr <<< "$_curLine"
	_curSchema_id=`echo $arr | cut -d " " -f 3`
	_schemasArr+=("$_curSchema_id")
    done

    for i in "${_schemasArr[@]}"
    do
	i="$(echo -e "${i}" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')"
	_schemas="$_schemas $i"
    done
    IFS=$OIFS
}

bashCompletion()
{
    local _cur _prev opts
    COMPREPLY=()
    _cur="${COMP_WORDS[COMP_CWORD]}"
    _prev="${COMP_WORDS[COMP_CWORD-1]}"
    _c=" "
    let "_c=(${COMP_CWORD}+$_gohanCommandPos) % 2"
    if [[ "${COMP_CWORD}" -lt $((_gohanCommandPos+1)) ]] ; then
	return 0
    fi
    if [[ "${COMP_WORDS[0]}" != "gohan" ]] &&  [[ "${COMP_WORDS[1]}" != "client" ]] ; then
	return 0
    fi
    _execOperation="${COMP_WORDS[0]} client";
    $_execOperation &> /dev/null;
    _exitCode=$?;
    if [[ _exitCode -ne 0 ]]; then
	return 0;
    fi
    if [[ `${COMP_WORDS[0]} client` == "Environment variable GOHAN_SERVICE_NAME needs to be set" ]] ; then
	return 0
    fi
    if [[ "${COMP_CWORD}" -eq $((_gohanCommandPos+1)) ]] ; then
	getAllSchemas
	COMPREPLY=( $(compgen -W "${_schemas}" -- ${_cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -eq $((_gohanCommandPos+2)) ]] ; then
	COMPREPLY=( $(compgen -W "${_commands}" -- ${_cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ _c -eq 1 ]] &&  [[ "${_cur}" == -* ]] ; then
	_schema_id="${COMP_WORDS[$((_gohanCommandPos+1))]}"
	getProperties $_schema_id
	IFS=$OIFS
	COMPREPLY=( $(compgen -W "${_named}" -- ${_cur} ) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ _c -eq 1 ]] && [[ "${_cur}" != -* ]] ; then
	_toShow=0
	_curCommand="${COMP_WORDS[3]}"
	[[ $__un_namedCommandWords =~ (^|[[:space:]])$_curCommand($|[[:space:]]) ]] && _toShow=1 || _toShow=0
	if [[ $_toShow == 0 ]]; then
            return 0
	fi
	_device="${COMP_WORDS[$((_gohanCommandPos+1))]}"
	getUn_namedProperties $_device
	IFS=$OIFS
	COMPREPLY=( $(compgen -W "${_un_named}" -- ${_cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ _c -eq 0 ]] && [[ "${_prev}" == "--fields" ]] ; then
	getProperties "${COMP_WORDS[$((_gohanCommandPos+1))]}"
	IFS=$OIFS
	COMPREPLY=( $(compgen -W "${_fields}" -- ${_cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ _c -eq 0 ]] && [[ "${_prev}" == "--verbosity" ]] ; then
	COMPREPLY=( $(compgen -W "${_verbosity}" -- ${_cur}) )
	return 0
    elif [[ "${COMP_CWORD}" -gt $((_gohanCommandPos+2)) ]] && [[ _c -eq 0 ]] && [[ "${_prev}" == "--output-format" ]] ; then
	COMPREPLY=( $(compgen -W "${_outputFormat}" -- ${_cur}) )
	return 0

    fi
}
complete -F bashCompletion gohan client

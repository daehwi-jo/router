#!/bin/bash

# $1 -> d option value
#prefix check
if [ -z "$1" ]
then
    echo -n "[Input] prefix : "
    read iprefix
else
    iprefix=$1
fi
prefix=${iprefix%/}

IFS=\/ read -ra NAMES <<< "$prefix";
root_name=${NAMES[ 1 ]}
plg_name=${NAMES[ 2 ]}
plugin_name=${NAMES[ 4 ]}

echo " #### $plugin_name:  start to run #### "

# for web directiory
cd  ${prefix%/}

if [ $root_name = "smartagent" -a $plg_name = "Plugins" ]
then
    if [ ${NAMES[ $(( ${#NAMES[*]} - 2 )) ]} = "DFA" ]
    then
        ${prefix%/}/$plugin_name -d ${prefix%/} &
    else
        ${prefix%/}/$plugin_name -d ${prefix%/} -e ${NAMES[ $(( ${#NAMES[*]} - 2 )) ]} &
    fi
else
    ${prefix%/}/$plugin_name -d ${prefix%/} &
fi

echo ""
echo ""
echo " #### $plugin_name:  check now state #### "
ps -ef | grep ${prefix%/}/$plugin_name; sleep 1
echo ""
echo ""

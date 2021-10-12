name=$1
if [[ X$1 == X ]];then
    echo "input server name to replace"
    exit 1
fi

sed -i "s/cron_framework/$name/g" conf/main.conf
mv cron_framework.go ${name}.go

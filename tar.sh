#! /bin/sh
filename=`date +%Y%m%d-%H%M`
tar zcvf dschat_$filename.tar.gz \
config.enc.yml \
DBConfig.enc.yml \
ServerConfig.yml \
deepseek-chat \
systemlist.yml \
templates/ \
run.sh \
tar.sh
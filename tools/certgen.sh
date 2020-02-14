#!/bin/bash

CAPATH=${CAPATH:-/etc/dgtwin/ca}
CASUBJECT=${CASUBJECT:-/C=CN/ST=Shaanxi/L=Xian/O=EDGEDEV/CN=edgedev.io}
CERTPATH=${CERTPATH:-/etc/dgtwin/certs}
CERTUBJECT=${CASUBJECT}
PASS_PHRASE="pass:cnj1990@sina.com.cn"
#subj 指定证书信息（国家、省份、城市、公司、CN（common name）管理员邮箱）

function genCA() {
	## create private key
	openssl genrsa -des3 -out ${CAPATH}/rootCA.key -passout ${PASS_PHRASE} 4096
	## create the cert.
	openssl req -x509 -new -nodes -key ${CAPATH}/rootCA.key -sha256 -days 3650 \
    -subj ${CASUBJECT} -passin ${PASS_PHRASE} -out ${CAPATH}/rootCA.crt	
}

## generate Cert request and cert.
function genCsrAndCert() {
	local name=$1

	openssl genrsa -out ${CERTPATH}/${name}.key 2048
	openssl req -new -key ${CERTPATH}/${name}.key -subj ${CERTUBJECT} -out ${CERTPATH}/${name}.csr
	openssl x509 -req -in ${CERTPATH}/${name}.csr -CA ${CAPATH}/rootCA.crt -CAkey ${CAPATH}/rootCA.key \
    -CAcreateserial -passin ${PASS_PHRASE} -out ${CERTPATH}/${name}.crt -days 3650 -sha256
}

function genCertAndkey() {
	if [ ! -d ${CAPATH} ]; then
        mkdir -p ${CAPATH}
    fi
    if [ ! -d ${CERTPATH} ]; then
        mkdir -p ${CERTPATH}
    fi

	## generate CA
	if [ ! -e ${CAPATH}/rootCA.key ] || [ ! -e ${CAPATH}/rootCA.crt ]; then
        genCA
    fi

	local name=$1
	genCsrAndCert ${name}
}

genCertAndkey edge

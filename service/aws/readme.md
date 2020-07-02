# AWS

## Iot

### 创建证书

[注册 CA 证书](https://cn-northwest-1.console.amazonaws.cn/iot/home?region=cn-northwest-1#/certificatehub)

创建证书与私钥
``` bash
openssl req -newkey rsa:2048 -nodes -keyout key.pem -x509 -days 365 -out certificate.pem
```

为私有密钥验证证书生成密钥对
``` bash
openssl genrsa -out verificationCert.key 2048
```

使用此注册代码创建 CSR
``` bash
openssl req -new -key verificationCert.key -out verificationCert.csr
```

将此注册代码放在公用名字段中
``` text
Country Name (2 letter code) [AU]:
State or Province Name (full name) [Some-State]:
Locality Name (eg, city) []:
Organization Name (eg, company) [Internet Widgits Pty Ltd]:
Organizational Unit Name (eg, section) []:
Common Name (e.g. server FQDN or YOUR name) []: e472a79b33b28904b592af65208bb7757ac56852ad236c999bd5ae4f0c30959f
Email Address []:
```

使用由 CA 私有密钥签名的 CSR 创建私有密钥验证证书
``` bash
openssl x509 -req -in verificationCert.csr -CA certificate.pem -CAkey key.pem -CAcreateserial -out verificationCert.crt -days 500 -sha256
```

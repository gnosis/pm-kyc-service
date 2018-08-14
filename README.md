# Prediction Markets KYC Service

## Installation
* It needs Go >= 1.10.3 https://golang.org/dl/
* Uses dep as dependency manager: https://github.com/golang/dep

```
# This will take a few minutes
dep ensure -v

# Fix to C bindings for secp256k1
go get github.com/ethereum/go-ethereum
cp -r \
  "${GOPATH:-$HOME/go}/src/github.com/ethereum/go-ethereum/crypto/secp256k1/libsecp256k1" \
  "vendor/github.com/ethereum/go-ethereum/crypto/secp256k1/"

# Use beego cli for easier development
go get github.com/beego/bee

```

## Run the project
```
bee run -downdoc=true -gendoc=true
```

## Test
```
go test tests/default_test.go -v
```

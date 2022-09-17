#!/bin/sh

curl -s https://raw.githubusercontent.com/asymptoter/gfg/master/main.go > .git/hooks/main.go

pushd .git/hooks &>/dev/null

go mod init &> /dev/null
go mod tidy
go build -o gfg
rm go.mod
rm main.go

if [[ ! -f "pre-push" ]]; then
	touch pre-push
	echo "#!/bin/sh" >> pre-push
	chmod +x pre-push	
fi

grep -Fxq ".git/hooks/gfg" pre-push || echo ".git/hooks/gfg" >> pre-push

popd &>/dev/null

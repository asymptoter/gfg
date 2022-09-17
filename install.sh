#!/bin/sh

GIT_REPOSITORY_ROOT=$1

curl https://raw.githubusercontent.com/asymptoter/gfg/master/main.go > "${GIT_REPOSITORY_ROOT}/.git/hooks/main.go"

pushd ${GIT_REPOSITORY_ROOT}/.git/hooks

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

echo ".git/hooks/gfg" >> pre-push

popd

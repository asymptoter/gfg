#!/bin/sh

git_hook=$1

curl -s https://raw.githubusercontent.com/asymptoter/gfg/master/main.go > .git/hooks/main.go

pushd .git/hooks &>/dev/null

go mod init &> /dev/null
go mod tidy
go build -o gfg
rm go.mod
rm main.go

if [[ ! -f "$git_hook" ]]; then
	touch $git_hook
	echo "#!/bin/sh" >> $git_hook
	chmod +x $git_hook
fi

grep -Fxq ".git/hooks/gfg" $git_hook || echo ".git/hooks/gfg" >> $git_hook

popd &>/dev/null

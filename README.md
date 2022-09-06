# About this project

This project can let you run go test on only modified packages and packages that depend on modified ones in a git repository. This could save you a lot of time instead of just running
```bash
go test ./...
```
before `git push`.

# Install

##### 1. Setup ENV used by `main.go` and `pre-push`
 
```bash
export GO_MOD_PATH=<path_of_go.mod>
export GO_MODULE_NAME=<name_of_go_module>
export BASE_BRANCH_NAME=<git_base_branch_name>
export GIT_REPOSITORY_ROOT=<git_repository_root_path>
```

##### 2. Download the `main.go` and `pre-push` then put them under the directory

```bash
$ curl https://github.com/asymptoter/gfg/blob/master/main.go > $GIT_REPOSITORY_ROOT/.git/hooks/main.go  
$ curl https://github.com/asymptoter/gfg/blob/master/pre-push > $GIT_REPOSITORY_ROOT/.git/hooks/pre-push  
```

##### 3. (optional) Add following line into `.gitignore`

```
$GO_MOD_PATH/.go_module_dependency_map
```

# Usage

```bash
$ git push
```

# Contributing

Feel free to open issues, or [email](asymptotion@gmail.com) to me ask anything about this project.

# TODO

1. Add more test cases.
2. Ignores packages without `_test.go` files.
3. Beautify console output.
4. Anything makes performance better.

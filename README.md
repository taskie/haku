# haku

## Install

```
go install github.com/taskie/haku/cmd/haku
```

## Usage

```
$ haku tmux showb &
$ curl localhost:8900
```

```
$ du -sch /* | haku -p | sort -rh &
$ curl localhost:8900
$ curl localhost:8900  # empty if -p (--persistent) mode is disabled
```

## License

Apache License 2.0

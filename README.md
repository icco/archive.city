# archive.city

```
$ wget https://raw.githubusercontent.com/jfrazelle/dotfiles/master/etc/docker/seccomp/chrome.json -O ~/chrome.json
$ docker run -it -p 8080:8080 --security-opt seccomp=$HOME/chrome.json icco/rendertron
```

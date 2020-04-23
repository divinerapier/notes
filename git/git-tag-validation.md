# Git Tag Validation In Makefile

## Makefile

``` makefile
# it is evaluated when is used (recursively expanded variable)
# https://ftp.gnu.org/old-gnu/Manuals/make-3.79.1/html_chapter/make_6.html#SEC59
git_tag = $(shell git describe --abbrev=0 --tags)
# Semantic versioning format https://semver.org/
tag_regex := ^v([0-9]+\.){2}[0-9]+$

build-image-prod:
ifeq ($(shell echo ${git_tag} | egrep "${tag_regex}"),)
	@echo "No Git tag selected. Are there tags?"
else
	@git checkout ${git_tag} -q
	@echo "Building image for production for Git tag $(git_tag)"
	docker build --target prod --tag trivago/ha-ci-api:$(git_tag) --file docker/api/Dockerfile .
endif
```

## Reference

[Git tag regex validation using Makefile](https://gist.github.com/jesugmz/a155b4a6686c4172048fabc6836c59e1)

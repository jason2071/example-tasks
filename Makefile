.PHONY: test test-unit test-e2e test-e2e-get-task-by-id test-e2e-update-task test-e2e-delete-task test-all

UNIT_PKGS := $(shell go list ./... | grep -v '/test/e2e$$')

test: test-all

test-unit:
	go test $(UNIT_PKGS) -v -count=1

test-e2e:
	go test ./test/e2e -v -count=1

test-e2e-get-task-by-id:
	go test ./test/e2e -run TestGetTaskByIDE2E -v -count=1

test-e2e-update-task:
	go test ./test/e2e -run TestUpdateTaskE2E -v -count=1

test-e2e-delete-task:
	go test ./test/e2e -run TestDeleteTaskE2E -v -count=1

test-all: test-unit test-e2e

#!/usr/bin/env bash
# Wrapper: passes -tags plus to the generator when PLUS env var is set.
exec go run ${PLUS:+-tags plus} ../generate/

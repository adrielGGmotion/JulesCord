#!/bin/bash
cat migrations/112_global_multipliers.up.sql 2>/dev/null || echo "112_global_multipliers.up.sql does not exist"
cat internal/bot/commands/multiplier.go 2>/dev/null || echo "multiplier.go does not exist"

`generate` now writes `.sops.yaml` atomically (temp file in the same directory, then rename), so an interrupted or crashed run can no longer leave a truncated file behind.

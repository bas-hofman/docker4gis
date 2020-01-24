#!/bin/bash
set -e

schema()
{
    pg.sh -c "set schema '${SCHEMA}'" "$@"
}

schema -f new_user.sql
schema -f change_password.sql
schema -f login.sql
schema -f save_password.sql
schema -f logout.sql

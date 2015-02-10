-- this function allows us to create indexes if they don't exist
create or replace function create_index_if_not_exists (t_name text, i_name text, index_sql text) returns void as $$
declare
  full_index_name varchar;
  schema_name varchar;
begin

full_index_name = t_name || '_' || i_name;
schema_name = 'public';

if not exists (
    select 1
    from   pg_class c
    join   pg_namespace n on n.oid = c.relnamespace
    where  c.relname = full_index_name
    and    n.nspname = schema_name
    ) then

    execute 'create index ' || full_index_name || ' on ' || schema_name || '.' || t_name || ' ' || index_sql;
end if;
end
$$
language plpgsql volatile;
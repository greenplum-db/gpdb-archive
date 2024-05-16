/* gpcontrib/gp_toolkit/gp_toolkit--1.5--1.6.sql */

-- complain if script is sourced in psql, rather than via ALTER EXTENSION
\echo Use "ALTER EXTENSION gp_toolkit UPDATE TO '1.6'" to load this file. \quit

ALTER TABLE gp_toolkit.gp_resgroup_config RENAME COLUMN memory_limit TO memory_quota;

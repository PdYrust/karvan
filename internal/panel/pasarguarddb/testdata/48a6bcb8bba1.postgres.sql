--
-- PostgreSQL database dump
--



SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: adminstatus; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.adminstatus AS ENUM (
    'active',
    'disabled',
    'limited'
);


--
-- Name: apikeystatus; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.apikeystatus AS ENUM (
    'active',
    'disabled'
);


--
-- Name: coretype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.coretype AS ENUM (
    'xray',
    'wg',
    'mtproto',
    'singbox'
);


--
-- Name: datalimitresetstrategy; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.datalimitresetstrategy AS ENUM (
    'no_reset',
    'day',
    'week',
    'month',
    'year'
);


--
-- Name: nodeconnectiontype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.nodeconnectiontype AS ENUM (
    'grpc',
    'rest'
);


--
-- Name: nodestatus; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.nodestatus AS ENUM (
    'connected',
    'connecting',
    'error',
    'disabled',
    'limited'
);


--
-- Name: proxyhostalpn; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.proxyhostalpn AS ENUM (
    'h3',
    'h3,h2',
    'h3,h2,http/1.1',
    'none',
    'h2',
    'http/1.1',
    'h2,http/1.1'
);


--
-- Name: proxyhostfingerprint; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.proxyhostfingerprint AS ENUM (
    'none',
    'chrome',
    'firefox',
    'safari',
    'ios',
    'android',
    'edge',
    '360',
    'qq',
    'random',
    'randomized',
    'randomizednoalpn',
    'unsafe'
);


--
-- Name: proxyhostsecurity; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.proxyhostsecurity AS ENUM (
    'inbound_default',
    'none',
    'tls'
);


--
-- Name: proxytypes; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.proxytypes AS ENUM (
    'VMess',
    'VLESS',
    'Trojan',
    'Shadowsocks'
);


--
-- Name: remindertype; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.remindertype AS ENUM (
    'expiration_date',
    'data_usage'
);


--
-- Name: userstatus; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.userstatus AS ENUM (
    'active',
    'disabled',
    'limited',
    'expired',
    'on_hold'
);


--
-- Name: userstatuscreate; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.userstatuscreate AS ENUM (
    'active',
    'on_hold'
);


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: admin_notification_reminders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admin_notification_reminders (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    admin_id bigint NOT NULL,
    type public.remindertype NOT NULL,
    threshold integer DEFAULT 0 NOT NULL
);


--
-- Name: admin_notification_reminders_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admin_notification_reminders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admin_notification_reminders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admin_notification_reminders_id_seq OWNED BY public.admin_notification_reminders.id;


--
-- Name: admin_roles; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admin_roles (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    is_owner boolean DEFAULT false NOT NULL,
    permissions jsonb NOT NULL,
    limits jsonb NOT NULL,
    features jsonb NOT NULL,
    access jsonb NOT NULL,
    created_at timestamp with time zone NOT NULL,
    disabled_when_limited boolean DEFAULT false NOT NULL,
    disconnect_users_when_limited boolean DEFAULT true NOT NULL,
    hwid json NOT NULL,
    disconnect_users_when_disabled boolean DEFAULT true NOT NULL
);


--
-- Name: admin_roles_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admin_roles_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admin_roles_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admin_roles_id_seq OWNED BY public.admin_roles.id;


--
-- Name: admin_usage_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admin_usage_logs (
    id bigint NOT NULL,
    admin_id bigint NOT NULL,
    used_traffic_at_reset bigint NOT NULL,
    reset_at timestamp with time zone NOT NULL
);


--
-- Name: admin_usage_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admin_usage_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admin_usage_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admin_usage_logs_id_seq OWNED BY public.admin_usage_logs.id;


--
-- Name: admins; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.admins (
    id bigint NOT NULL,
    username character varying(34) NOT NULL,
    hashed_password character varying(128) NOT NULL,
    created_at timestamp with time zone NOT NULL,
    password_reset_at timestamp with time zone,
    telegram_id bigint,
    discord_webhook character varying(1024),
    used_traffic bigint DEFAULT '0'::bigint NOT NULL,
    sub_template character varying(1024),
    sub_domain character varying(256),
    profile_title character varying(512),
    support_url character varying(1024),
    notification_enable jsonb,
    note character varying(500),
    role_id bigint NOT NULL,
    permission_overrides jsonb,
    status public.adminstatus DEFAULT 'active'::public.adminstatus NOT NULL,
    data_limit bigint,
    last_status_change timestamp with time zone,
    custom_variables json
);


--
-- Name: admins_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.admins_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: admins_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.admins_id_seq OWNED BY public.admins.id;


--
-- Name: alembic_version; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.alembic_version (
    version_num character varying(32) NOT NULL
);


--
-- Name: api_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.api_keys (
    id bigint NOT NULL,
    admin_id bigint NOT NULL,
    name character varying(128) NOT NULL,
    note character varying(512),
    key_hash character varying(128) NOT NULL,
    api_key_trimmed character varying(16) NOT NULL,
    permissions json NOT NULL,
    created_at timestamp with time zone NOT NULL,
    expire_date timestamp with time zone,
    revoked_at timestamp with time zone,
    status public.apikeystatus DEFAULT 'active'::public.apikeystatus NOT NULL,
    inherit_permissions boolean DEFAULT true NOT NULL
);


--
-- Name: api_keys_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.api_keys_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: api_keys_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.api_keys_id_seq OWNED BY public.api_keys.id;


--
-- Name: client_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.client_templates (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    template_type character varying(32) NOT NULL,
    content text NOT NULL,
    is_default boolean DEFAULT false NOT NULL,
    is_system boolean DEFAULT false NOT NULL
);


--
-- Name: client_templates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.client_templates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: client_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.client_templates_id_seq OWNED BY public.client_templates.id;


--
-- Name: core_configs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.core_configs (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    name character varying(256) NOT NULL,
    config json NOT NULL,
    exclude_inbound_tags character varying(2048),
    fallbacks_inbound_tags character varying(2048),
    type public.coretype DEFAULT 'xray'::public.coretype NOT NULL
);


--
-- Name: core_configs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.core_configs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: core_configs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.core_configs_id_seq OWNED BY public.core_configs.id;


--
-- Name: groups; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.groups (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    is_disabled boolean DEFAULT false NOT NULL
);


--
-- Name: groups_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.groups_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: groups_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.groups_id_seq OWNED BY public.groups.id;


--
-- Name: hosts; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.hosts (
    id bigint NOT NULL,
    remark character varying(256) NOT NULL,
    address character varying(256) NOT NULL,
    port integer,
    inbound_tag character varying(256),
    sni character varying(1000),
    host character varying(1000),
    security public.proxyhostsecurity DEFAULT 'inbound_default'::public.proxyhostsecurity NOT NULL,
    fingerprint public.proxyhostfingerprint DEFAULT 'none'::public.proxyhostfingerprint NOT NULL,
    allowinsecure boolean,
    is_disabled boolean,
    path character varying(256),
    random_user_agent boolean DEFAULT false NOT NULL,
    alpn character varying(14) DEFAULT 'none'::public.proxyhostalpn,
    use_sni_as_host boolean DEFAULT false NOT NULL,
    priority integer NOT NULL,
    http_headers json,
    transport_settings json,
    mux_settings json,
    noise_settings json,
    fragment_settings json,
    status character varying(60) DEFAULT ''::character varying,
    vless_route character varying(4),
    pinned_peer_cert_sha256 character varying(128),
    verify_peer_cert_by_name character varying(1000),
    wireguard_overrides json,
    subscription_templates json,
    final_mask_settings json,
    cipher_suites character varying(1024),
    ech json
);


--
-- Name: hosts_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.hosts_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: hosts_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.hosts_id_seq OWNED BY public.hosts.id;


--
-- Name: inbounds; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inbounds (
    id bigint NOT NULL,
    tag character varying(256) NOT NULL
);


--
-- Name: inbounds_groups_association; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.inbounds_groups_association (
    inbound_id bigint NOT NULL,
    group_id bigint NOT NULL
);


--
-- Name: inbounds_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.inbounds_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: inbounds_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.inbounds_id_seq OWNED BY public.inbounds.id;


--
-- Name: jwt; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.jwt (
    id bigint NOT NULL,
    secret_key character varying(64) NOT NULL
);


--
-- Name: jwt_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.jwt_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: jwt_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.jwt_id_seq OWNED BY public.jwt.id;


--
-- Name: next_plans; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.next_plans (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    data_limit bigint NOT NULL,
    expire integer,
    add_remaining_traffic boolean DEFAULT false NOT NULL,
    user_template_id bigint
);


--
-- Name: next_plans_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.next_plans_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: next_plans_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.next_plans_id_seq OWNED BY public.next_plans.id;


--
-- Name: node_stats; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.node_stats (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    node_id bigint NOT NULL,
    mem_total bigint NOT NULL,
    mem_used bigint NOT NULL,
    cpu_cores integer NOT NULL,
    cpu_usage double precision NOT NULL,
    incoming_bandwidth_speed bigint NOT NULL,
    outgoing_bandwidth_speed bigint NOT NULL
);


--
-- Name: node_stats_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.node_stats_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: node_stats_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.node_stats_id_seq OWNED BY public.node_stats.id;


--
-- Name: node_usage_reset_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.node_usage_reset_logs (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    node_id bigint NOT NULL,
    uplink bigint NOT NULL,
    downlink bigint NOT NULL
);


--
-- Name: node_usage_reset_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.node_usage_reset_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: node_usage_reset_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.node_usage_reset_logs_id_seq OWNED BY public.node_usage_reset_logs.id;


--
-- Name: node_usages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.node_usages (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    node_id bigint,
    uplink bigint NOT NULL,
    downlink bigint NOT NULL
);


--
-- Name: node_usages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.node_usages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: node_usages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.node_usages_id_seq OWNED BY public.node_usages.id;


--
-- Name: node_user_usages; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.node_user_usages (
    id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    user_id bigint NOT NULL,
    node_id bigint,
    used_traffic bigint NOT NULL
);


--
-- Name: node_user_usages_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.node_user_usages_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: node_user_usages_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.node_user_usages_id_seq OWNED BY public.node_user_usages.id;


--
-- Name: nodes; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.nodes (
    id bigint NOT NULL,
    name character varying(256) NOT NULL,
    address character varying(256) NOT NULL,
    port integer NOT NULL,
    status public.nodestatus NOT NULL,
    last_status_change timestamp with time zone,
    message character varying(1024),
    created_at timestamp with time zone NOT NULL,
    uplink bigint NOT NULL,
    downlink bigint NOT NULL,
    xray_version character varying(32),
    usage_coefficient double precision DEFAULT 1.0 NOT NULL,
    node_version character varying(32),
    connection_type public.nodeconnectiontype DEFAULT 'grpc'::public.nodeconnectiontype NOT NULL,
    server_ca character varying(2048) DEFAULT ''::character varying NOT NULL,
    keep_alive integer DEFAULT 0 NOT NULL,
    core_config_id bigint,
    api_key character varying(36),
    data_limit bigint DEFAULT 0 NOT NULL,
    data_limit_reset_strategy public.datalimitresetstrategy DEFAULT 'no_reset'::public.datalimitresetstrategy NOT NULL,
    reset_time integer DEFAULT '-1'::integer NOT NULL,
    default_timeout integer DEFAULT 10 NOT NULL,
    internal_timeout integer DEFAULT 15 NOT NULL,
    api_port integer DEFAULT 62051 NOT NULL,
    proxy_url character varying(256)
);


--
-- Name: nodes_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.nodes_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: nodes_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.nodes_id_seq OWNED BY public.nodes.id;


--
-- Name: notification_reminders; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.notification_reminders (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    type public.remindertype NOT NULL,
    expires_at timestamp with time zone,
    created_at timestamp with time zone NOT NULL,
    threshold integer
);


--
-- Name: notification_reminders_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.notification_reminders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: notification_reminders_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.notification_reminders_id_seq OWNED BY public.notification_reminders.id;


--
-- Name: settings; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.settings (
    id bigint NOT NULL,
    telegram json NOT NULL,
    webhook json NOT NULL,
    notification_settings json NOT NULL,
    notification_enable json NOT NULL,
    subscription json NOT NULL,
    general json NOT NULL,
    hwid json NOT NULL
);


--
-- Name: settings_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.settings_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: settings_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.settings_id_seq OWNED BY public.settings.id;


--
-- Name: system; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.system (
    id bigint NOT NULL,
    uplink bigint NOT NULL,
    downlink bigint NOT NULL
);


--
-- Name: system_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.system_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: system_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.system_id_seq OWNED BY public.system.id;


--
-- Name: temp_keys; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.temp_keys (
    key character varying(36) NOT NULL,
    action character varying(32) NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    used_at timestamp with time zone,
    used_by_ip character varying(45)
);


--
-- Name: template_group_association; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.template_group_association (
    user_template_id bigint,
    group_id bigint
);


--
-- Name: user_hwids; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_hwids (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    hwid character varying(256) NOT NULL,
    device_os character varying(256),
    os_version character varying(128),
    device_model character varying(256),
    created_at timestamp with time zone NOT NULL,
    last_used_at timestamp with time zone NOT NULL
);


--
-- Name: user_hwids_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_hwids_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_hwids_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_hwids_id_seq OWNED BY public.user_hwids.id;


--
-- Name: user_subscription_updates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_subscription_updates (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    created_at timestamp with time zone NOT NULL,
    user_agent character varying(512) NOT NULL,
    ip character varying(64),
    hwid character varying(256)
);


--
-- Name: user_subscription_updates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_subscription_updates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_subscription_updates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_subscription_updates_id_seq OWNED BY public.user_subscription_updates.id;


--
-- Name: user_templates; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_templates (
    id bigint NOT NULL,
    name character varying(64) NOT NULL,
    data_limit bigint NOT NULL,
    expire_duration bigint NOT NULL,
    username_prefix character varying(20),
    username_suffix character varying(20),
    extra_settings json,
    status public.userstatuscreate DEFAULT 'active'::public.userstatuscreate NOT NULL,
    reset_usages boolean DEFAULT false NOT NULL,
    on_hold_timeout integer,
    data_limit_reset_strategy public.datalimitresetstrategy DEFAULT 'no_reset'::public.datalimitresetstrategy NOT NULL,
    is_disabled boolean DEFAULT false NOT NULL,
    hwid_limit bigint
);


--
-- Name: user_templates_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_templates_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_templates_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_templates_id_seq OWNED BY public.user_templates.id;


--
-- Name: user_usage_logs; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.user_usage_logs (
    id bigint NOT NULL,
    user_id bigint,
    used_traffic_at_reset bigint NOT NULL,
    reset_at timestamp with time zone NOT NULL
);


--
-- Name: user_usage_logs_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.user_usage_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: user_usage_logs_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.user_usage_logs_id_seq OWNED BY public.user_usage_logs.id;


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    username character varying(128) NOT NULL COLLATE pg_catalog."C",
    status public.userstatus NOT NULL,
    used_traffic bigint NOT NULL,
    data_limit bigint,
    created_at timestamp with time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    admin_id bigint,
    data_limit_reset_strategy public.datalimitresetstrategy DEFAULT 'no_reset'::public.datalimitresetstrategy NOT NULL,
    sub_revoked_at timestamp with time zone,
    note character varying(500),
    online_at timestamp with time zone,
    edit_at timestamp with time zone,
    on_hold_timeout timestamp with time zone,
    on_hold_expire_duration bigint,
    auto_delete_in_days integer,
    last_status_change timestamp with time zone,
    expire timestamp with time zone,
    proxy_settings json DEFAULT '{}'::jsonb NOT NULL,
    hwid_limit bigint
);


--
-- Name: users_groups_association; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users_groups_association (
    user_id bigint NOT NULL,
    groups_id bigint NOT NULL
);


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: users_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.users_id_seq OWNED BY public.users.id;


--
-- Name: wireguard_subnets; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.wireguard_subnets (
    id bigint NOT NULL,
    network character varying(64) NOT NULL,
    next_offset bigint DEFAULT '1'::bigint NOT NULL,
    free_offsets json NOT NULL
);


--
-- Name: wireguard_subnets_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.wireguard_subnets_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- Name: wireguard_subnets_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.wireguard_subnets_id_seq OWNED BY public.wireguard_subnets.id;


--
-- Name: admin_notification_reminders id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_notification_reminders ALTER COLUMN id SET DEFAULT nextval('public.admin_notification_reminders_id_seq'::regclass);


--
-- Name: admin_roles id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_roles ALTER COLUMN id SET DEFAULT nextval('public.admin_roles_id_seq'::regclass);


--
-- Name: admin_usage_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_usage_logs ALTER COLUMN id SET DEFAULT nextval('public.admin_usage_logs_id_seq'::regclass);


--
-- Name: admins id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admins ALTER COLUMN id SET DEFAULT nextval('public.admins_id_seq'::regclass);


--
-- Name: api_keys id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys ALTER COLUMN id SET DEFAULT nextval('public.api_keys_id_seq'::regclass);


--
-- Name: client_templates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_templates ALTER COLUMN id SET DEFAULT nextval('public.client_templates_id_seq'::regclass);


--
-- Name: core_configs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.core_configs ALTER COLUMN id SET DEFAULT nextval('public.core_configs_id_seq'::regclass);


--
-- Name: groups id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groups ALTER COLUMN id SET DEFAULT nextval('public.groups_id_seq'::regclass);


--
-- Name: hosts id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hosts ALTER COLUMN id SET DEFAULT nextval('public.hosts_id_seq'::regclass);


--
-- Name: inbounds id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbounds ALTER COLUMN id SET DEFAULT nextval('public.inbounds_id_seq'::regclass);


--
-- Name: jwt id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jwt ALTER COLUMN id SET DEFAULT nextval('public.jwt_id_seq'::regclass);


--
-- Name: next_plans id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.next_plans ALTER COLUMN id SET DEFAULT nextval('public.next_plans_id_seq'::regclass);


--
-- Name: node_stats id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_stats ALTER COLUMN id SET DEFAULT nextval('public.node_stats_id_seq'::regclass);


--
-- Name: node_usage_reset_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usage_reset_logs ALTER COLUMN id SET DEFAULT nextval('public.node_usage_reset_logs_id_seq'::regclass);


--
-- Name: node_usages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usages ALTER COLUMN id SET DEFAULT nextval('public.node_usages_id_seq'::regclass);


--
-- Name: node_user_usages id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_user_usages ALTER COLUMN id SET DEFAULT nextval('public.node_user_usages_id_seq'::regclass);


--
-- Name: nodes id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nodes ALTER COLUMN id SET DEFAULT nextval('public.nodes_id_seq'::regclass);


--
-- Name: notification_reminders id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_reminders ALTER COLUMN id SET DEFAULT nextval('public.notification_reminders_id_seq'::regclass);


--
-- Name: settings id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings ALTER COLUMN id SET DEFAULT nextval('public.settings_id_seq'::regclass);


--
-- Name: system id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system ALTER COLUMN id SET DEFAULT nextval('public.system_id_seq'::regclass);


--
-- Name: user_hwids id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_hwids ALTER COLUMN id SET DEFAULT nextval('public.user_hwids_id_seq'::regclass);


--
-- Name: user_subscription_updates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_subscription_updates ALTER COLUMN id SET DEFAULT nextval('public.user_subscription_updates_id_seq'::regclass);


--
-- Name: user_templates id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_templates ALTER COLUMN id SET DEFAULT nextval('public.user_templates_id_seq'::regclass);


--
-- Name: user_usage_logs id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_usage_logs ALTER COLUMN id SET DEFAULT nextval('public.user_usage_logs_id_seq'::regclass);


--
-- Name: users id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users ALTER COLUMN id SET DEFAULT nextval('public.users_id_seq'::regclass);


--
-- Name: wireguard_subnets id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wireguard_subnets ALTER COLUMN id SET DEFAULT nextval('public.wireguard_subnets_id_seq'::regclass);


--
-- Data for Name: admin_notification_reminders; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: admin_roles; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.admin_roles VALUES (2, 'administrator', false, '{"cores": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "hosts": {"read": true, "create": true, "update": true}, "hwids": {"read": true, "delete": true}, "nodes": {"logs": true, "read": true, "stats": true, "create": true, "delete": true, "update": true, "reconnect": true, "read_simple": true, "update_core": true}, "users": {"read": {"scope": 2}, "create": true, "delete": {"scope": 2}, "update": {"scope": 2}, "set_owner": true, "revoke_sub": {"scope": 2}, "read_simple": {"scope": 2}, "reset_usage": {"scope": 2}, "activate_next_plan": {"scope": 2}}, "admins": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true, "reset_usage": true}, "groups": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "system": {"read": true}, "api_keys": {"read": {"scope": 2}, "create": true, "delete": {"scope": 2}, "update": {"scope": 2}, "read_simple": {"scope": 2}}, "settings": {"read": true, "update": true, "read_general": true}, "templates": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "admin_roles": {"read": true, "read_simple": true}, "client_templates": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}}', '{"max_users": null, "expire_max": null, "expire_min": null, "data_limit_max": null, "data_limit_min": null, "max_hwid_per_user": null, "min_hwid_per_user": null}', '{"can_use_next_plan": true, "can_use_reset_strategy": true}', '{"require_template": false, "allowed_group_ids": null, "allowed_template_ids": null}', '2026-10-16 13:41:28.709633+00', false, true, '{"mode": "use_global", "forced": false, "enabled": true, "max_limit": null, "min_limit": null, "fallback_limit": null}', true);
INSERT INTO public.admin_roles VALUES (1, 'owner', true, '{"cores": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "hosts": {"read": true, "create": true, "update": true}, "hwids": {"read": true, "delete": true}, "nodes": {"logs": true, "read": true, "stats": true, "create": true, "delete": true, "update": true, "reconnect": true, "read_simple": true, "update_core": true}, "users": {"read": {"scope": 2}, "create": true, "delete": {"scope": 2}, "update": {"scope": 2}, "set_owner": true, "revoke_sub": {"scope": 2}, "read_simple": {"scope": 2}, "reset_usage": {"scope": 2}, "activate_next_plan": {"scope": 2}}, "admins": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true, "reset_usage": true}, "groups": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "system": {"read": true}, "api_keys": {"read": {"scope": 2}, "create": true, "delete": {"scope": 2}, "update": {"scope": 2}, "read_simple": {"scope": 2}}, "settings": {"read": true, "update": true, "read_general": true}, "templates": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "admin_roles": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}, "client_templates": {"read": true, "create": true, "delete": true, "update": true, "read_simple": true}}', '{"max_users": null, "expire_max": null, "expire_min": null, "data_limit_max": null, "data_limit_min": null, "max_hwid_per_user": null, "min_hwid_per_user": null}', '{"can_use_next_plan": true, "can_use_reset_strategy": true}', '{"require_template": false, "allowed_group_ids": null, "allowed_template_ids": null}', '2026-10-16 13:41:28.709633+00', false, true, '{"mode": "use_global", "forced": false, "enabled": true, "max_limit": null, "min_limit": null, "fallback_limit": null}', true);
INSERT INTO public.admin_roles VALUES (3, 'operator', false, '{"hwids": {"read": true, "delete": true}, "users": {"read": {"scope": 1}, "create": true, "delete": {"scope": 1}, "update": {"scope": 1}, "set_owner": null, "revoke_sub": {"scope": 1}, "read_simple": {"scope": 1}, "reset_usage": {"scope": 1}, "activate_next_plan": {"scope": 1}}, "groups": {"read": false, "create": false, "delete": false, "update": false, "read_simple": true}, "system": {"read": true}, "api_keys": {"read": {"scope": 1}, "delete": {"scope": 1}, "update": {"scope": 1}, "read_simple": {"scope": 1}}, "settings": {"read": false, "update": false, "read_general": true}, "templates": {"read": true, "create": false, "delete": false, "update": false, "read_simple": true}}', '{"max_users": null, "expire_max": null, "expire_min": null, "data_limit_max": null, "data_limit_min": null, "max_hwid_per_user": null, "min_hwid_per_user": null}', '{"can_use_next_plan": true, "can_use_reset_strategy": true}', '{"require_template": false, "allowed_group_ids": null, "allowed_template_ids": null}', '2026-10-16 13:41:28.709633+00', false, true, '{"mode": "use_global", "forced": false, "enabled": true, "max_limit": null, "min_limit": null, "fallback_limit": null}', true);


--
-- Data for Name: admin_usage_logs; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: admins; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: alembic_version; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.alembic_version VALUES ('48a6bcb8bba1');


--
-- Data for Name: api_keys; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: client_templates; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.client_templates VALUES (1, 'Default Clash Subscription', 'clash_subscription', 'mode: rule
mixed-port: 7890
ipv6: true

tun:
  enable: true
  stack: mixed
  dns-hijack:
    - "any:53"
  auto-route: true
  auto-detect-interface: true
  strict-route: true

dns:
  enable: true
  listen: :1053
  ipv6: true
  nameserver:
    - ''https://1.1.1.1/dns-query#PROXY''
  proxy-server-nameserver:
    - ''8.8.8.8''
    - ''1.1.1.1''

sniffer:
  enable: true
  override-destination: true
  sniff:
    HTTP:
      ports: [80, 8080-8880]
    TLS:
      ports: [443, 8443]
    QUIC:
      ports: [443, 8443]

{{ conf | except("proxy-groups", "port", "mode", "rules") | yaml }}

proxy-groups:
- name: ''PROXY''
  type: ''select''
  proxies:
  - ''Fastest''
  {{ proxy_remarks | yaml | indent(2) }}

- name: ''Fastest''
  type: ''url-test''
  proxies:
  {{ proxy_remarks | yaml | indent(2) }}

rules:
  - MATCH,PROXY', true, true);
INSERT INTO public.client_templates VALUES (2, 'Default Xray Subscription', 'xray_subscription', '{
  "log": {
    "access": "",
    "error": "",
    "loglevel": "warning"
  },
 "policy": {
    "system": {
        "statsOutboundDownlink": true,
        "statsOutboundUplink": true
    },
	"levels": {
     "8": {
       "connIdle": 300,
       "downlinkOnly": 1,
       "handshake": 4,
       "uplinkOnly": 1
     }
   }
   },
  "inbounds": [
    {
      "tag": "socks",
      "port": 10808,
      "listen": "0.0.0.0",
      "protocol": "socks",
      "sniffing": {
        "enabled": true,
        "destOverride": [
          "http",
          "tls"
        ],
        "routeOnly": false
      },
      "settings": {
        "auth": "noauth",
        "udp": true,
        "allowTransparent": false
      }
    },
    {
      "tag": "http",
      "port": 10809,
      "listen": "0.0.0.0",
      "protocol": "http",
      "sniffing": {
        "enabled": true,
        "destOverride": [
          "http",
          "tls"
        ],
        "routeOnly": false
      },
      "settings": {
        "auth": "noauth",
        "udp": true,
        "allowTransparent": false
      }
    }
  ],
  "outbounds": [
      {
        "protocol": "freedom",
        "tag": "DIRECT"
      },
      {
        "protocol": "blackhole",
        "tag": "BLOCK"
      }
    ],
  "dns": {
    "servers": [
      "1.1.1.1",
      "8.8.8.8"
    ]
  },
  "routing": {
    "domainStrategy": "AsIs",
    "rules": []
  }
}', true, true);
INSERT INTO public.client_templates VALUES (3, 'Default Singbox Subscription', 'singbox_subscription', '{
  "log": {
    "level": "warn",
    "timestamp": false
  },
  "dns": {
    "servers": [
      {
        "type": "udp",
        "tag": "dns-remote",
        "server": "1.1.1.2",
        "detour": "proxy"
      },
      {
        "type": "local",
        "tag": "dns-local"
      }
    ],
    "final": "dns-remote"
  },
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "interface_name": "sing-tun",
      "address": [
        "172.19.0.1/30",
        "fdfe:dcba:9876::1/126"
      ],
      "auto_route": true,
      "route_exclude_address": [
        "192.168.0.0/16",
        "10.0.0.0/8",
        "169.254.0.0/16",
        "172.16.0.0/12",
        "fe80::/10",
        "fc00::/7"
      ]
    }
  ],
  "outbounds": [
    {
      "type": "selector",
      "tag": "proxy",
      "outbounds": null,
      "interrupt_exist_connections": true
    },
    {
      "type": "urltest",
      "tag": "Best Latency",
      "outbounds": null
    },
    {
      "type": "direct",
      "tag": "direct"
    }
  ],
  "route": {
    "rules": [
      {
        "inbound": "tun-in",
        "action": "sniff"
      },
      {
        "protocol": "dns",
        "action": "hijack-dns"
      }
    ],
    "final": "proxy",
    "auto_detect_interface": true,
    "override_android_vpn": true
  },
  "experimental": {
    "cache_file": {
      "enabled": true,
      "store_dns": true
    }
  }
}', true, true);
INSERT INTO public.client_templates VALUES (4, 'Default User-Agent Template', 'user_agent', '{
  "list":[
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36 Edg/125.0.0.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/106.0.0.0 Safari/537.36 PageSpeedPlus/1.0.0",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:126.0) Gecko/20100101 Firefox/126.0",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_4_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4.1 Safari/605.1.15",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 11.6; rv:92.0) Gecko/20100101 Firefox/92.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.14 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:126.0) Gecko/20100101 Firefox/126.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Safari/605.1.15",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/109.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/25.0 Chrome/121.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:125.0) Gecko/20100101 Firefox/125.0",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:125.0) Gecko/20100101 Firefox/125.0",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.16 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/89.0.4389.114 Safari/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 Edg/123.0.0.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.3 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36 Edg/122.0.0.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2.1 Safari/605.1.15",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.12 Chrome/120.0.6099.283 Electron/28.2.3 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.16 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/123.0.0.0 Safari/537.36 OPR/109.0.0.0",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/81.0.4044.129 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/116.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Safari/605.1.15",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.13 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 6.1; Win64; x64; rv:109.0) Gecko/20100101 Firefox/115.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_3 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.1 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_3_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3.1 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:124.0) Gecko/20100101 Firefox/124.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.3 Safari/605.1.15",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.4.13 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.4 Safari/605.1.15",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/15.6.1 Safari/605.1.15",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_1_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.1 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 17_5 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.5 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.12 Chrome/120.0.6099.283 Electron/28.2.3 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.1.2 Safari/605.1.15",
    "Mozilla/5.0 (iPhone; CPU iPhone OS 16_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.2 Mobile/15E148 Safari/604.1",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.5 Safari/605.1.15",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36 Edg/121.0.0.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:124.0) Gecko/20100101 Firefox/124.0",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/118.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) obsidian/1.5.3 Chrome/114.0.5735.289 Electron/25.8.1 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.67 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.3 Safari/605.1.15",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/122.0.0.0 Safari/537.36",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/124.0.0.0 Safari/537.36 Edg/124.0.0.0",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/117.0.0.0 Mobile Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/114.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Linux; Android 10; K) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/24.0 Chrome/117.0.0.0 Mobile Safari/537.36"
  ]
}', true, true);
INSERT INTO public.client_templates VALUES (5, 'Default gRPC User-Agent Template', 'grpc_user_agent', '{
    "list": [
        "grpc-dotnet/2.41.0 (.NET 6.0.1; CLR 6.0.1; net6.0; windows; x64)",
        "grpc-dotnet/2.41.0 (.NET 6.0.0-preview.7.21377.19; CLR 6.0.0; net6.0; osx; x64)",
        "grpc-dotnet/2.41.0 (Mono 6.12.0.140; CLR 4.0.30319; netstandard2.0; osx; x64)",
        "grpc-dotnet/2.41.0 (.NET 6.0.0-rc.1.21380.1; CLR 6.0.0; net6.0; linux; arm64)",
        "grpc-dotnet/2.41.0 (.NET 5.0.8; CLR 5.0.8; net5.0; linux; arm64)",
        "grpc-dotnet/2.41.0 (.NET Core; CLR 3.1.4; netstandard2.1; linux; arm64)",
        "grpc-dotnet/2.41.0 (.NET Framework; CLR 4.0.30319.42000; netstandard2.0; windows; x86)",
        "grpc-dotnet/2.41.0 (.NET 6.0.0-rc.1.21380.1; CLR 6.0.0; net6.0; windows; x64)",
        "grpc-python-asyncio/1.62.1 grpc-c/39.0.0 (linux; chttp2)",
        "grpc-go/1.58.1",
        "grpc-java-okhttp/1.55.1",
        "grpc-node/1.7.1 grpc-c/1.7.1 (osx; chttp2)",
        "grpc-node/1.24.2 grpc-c/8.0.0 (linux; chttp2; ganges)",
        "grpc-c++/1.16.0 grpc-c/6.0.0 (linux; nghttp2; hw)",
        "grpc-node/1.19.0 grpc-c/7.0.0 (linux; chttp2; gold)",
        "grpc-ruby/1.62.0 grpc-c/39.0.0 (osx; chttp2)]"
    ]
}', true, true);


--
-- Data for Name: core_configs; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.core_configs VALUES (1, '2026-10-16 13:41:25.726906+00', 'Default Core Config', '{"policy": {"levels": {"0": {"statsUserOnline": true}}}, "log": {"loglevel": "warning"}, "inbounds": [{"tag": "Shadowsocks TCP", "listen": "0.0.0.0", "port": 1080, "protocol": "shadowsocks", "settings": {"clients": [], "network": "tcp,udp"}}], "outbounds": [{"protocol": "freedom", "tag": "DIRECT"}, {"protocol": "blackhole", "tag": "BLOCK"}], "routing": {"rules": [{"ip": ["geoip:private"], "outboundTag": "BLOCK", "type": "field"}]}}', '', '', 'xray');


--
-- Data for Name: groups; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: hosts; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: inbounds; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: inbounds_groups_association; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: jwt; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.jwt VALUES (1, '705bcf7887550e556229cfd8c4a3f6237a0787ca6591ed46512da8f85a342b80');


--
-- Data for Name: next_plans; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: node_stats; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: node_usage_reset_logs; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: node_usages; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: node_user_usages; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: nodes; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: notification_reminders; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: settings; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.settings VALUES (1, '{"enable": false, "token": null, "webhook_url": null, "webhook_secret": null, "proxy_url": null}', '{"enable": false, "webhooks": [], "days_left": [3], "usage_percent": [80], "timeout": 180, "recurrent": 3, "proxy_url": null}', '{"notify_telegram": false, "notify_discord": false, "telegram_api_token": null, "telegram_chat_id": null, "telegram_topic_id": null, "discord_webhook_url": null, "proxy_url": null, "max_retries": 3}', '{"admin": {"create": true, "modify": true, "delete": true, "reset_usage": true, "login": true}, "core": {"create": true, "modify": true, "delete": true}, "group": {"create": true, "modify": true, "delete": true}, "host": {"create": true, "modify": true, "delete": true, "modify_hosts": true}, "node": {"create": true, "modify": true, "delete": true, "connect": true, "error": true}, "user": {"create": true, "modify": true, "delete": true, "status_change": true, "reset_data_usage": true, "data_reset_by_next": true, "subscription_revoked": true}, "user_template": {"create": true, "modify": true, "delete": true}, "days_left": true, "percentage_reached": true}', '{"url_prefix": "", "update_interval": "12", "support_url": "https://t.me/", "profile_title": "Subscription", "host_status_filter": true, "randomize_order": false, "rules": [{"pattern": "^(?:FlClashX?|Flowvy|[Cc]lash(?:-(?:[Vv]erge|nyanpasu)|X [Mm]eta|-?[Mm]eta)|[Kk]oala-[Cc]lash|[Mm](?:urge|ihomo)|prizrak-box|clash\\.meta)", "target": "clash_meta"}, {"pattern": "^([Cc]lash|[Ss]tash)", "target": "clash"}, {"pattern": "^(SFA|SFI|SFM|SFT|[Kk]aring|[Hh]iddify[Nn]ext)|.*[Ss]ing[\\-b]?ox.*", "target": "sing_box"}, {"pattern": "^(SS|SSR|SSD|SSS|Outline|Shadowsocks|SSconf)", "target": "outline"}, {"pattern": "^[Ii]n[Hh]ive", "target": "xray"}, {"pattern": "^.*", "target": "links_base64"}], "manual_sub_request": {"links": true, "links_base64": true, "xray": true, "sing_box": true, "clash": true, "clash_meta": true, "outline": true}}', '{"default_flow": "", "default_method": "chacha20-ietf-poly1305"}', '{}');


--
-- Data for Name: system; Type: TABLE DATA; Schema: public; Owner: -
--

INSERT INTO public.system VALUES (1, 0, 0);


--
-- Data for Name: temp_keys; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: template_group_association; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_hwids; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_subscription_updates; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_templates; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: user_usage_logs; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: users_groups_association; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Data for Name: wireguard_subnets; Type: TABLE DATA; Schema: public; Owner: -
--



--
-- Name: admin_notification_reminders_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.admin_notification_reminders_id_seq', 1, false);


--
-- Name: admin_roles_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.admin_roles_id_seq', 3, true);


--
-- Name: admin_usage_logs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.admin_usage_logs_id_seq', 1, false);


--
-- Name: admins_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.admins_id_seq', 1, false);


--
-- Name: api_keys_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.api_keys_id_seq', 1, false);


--
-- Name: client_templates_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.client_templates_id_seq', 5, true);


--
-- Name: core_configs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.core_configs_id_seq', 1, true);


--
-- Name: groups_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.groups_id_seq', 1, false);


--
-- Name: hosts_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.hosts_id_seq', 1, false);


--
-- Name: inbounds_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.inbounds_id_seq', 1, false);


--
-- Name: jwt_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.jwt_id_seq', 1, false);


--
-- Name: next_plans_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.next_plans_id_seq', 1, false);


--
-- Name: node_stats_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.node_stats_id_seq', 1, false);


--
-- Name: node_usage_reset_logs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.node_usage_reset_logs_id_seq', 1, false);


--
-- Name: node_usages_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.node_usages_id_seq', 1, false);


--
-- Name: node_user_usages_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.node_user_usages_id_seq', 1, false);


--
-- Name: nodes_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.nodes_id_seq', 1, false);


--
-- Name: notification_reminders_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.notification_reminders_id_seq', 1, false);


--
-- Name: settings_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.settings_id_seq', 1, true);


--
-- Name: system_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.system_id_seq', 1, false);


--
-- Name: user_hwids_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.user_hwids_id_seq', 1, false);


--
-- Name: user_subscription_updates_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.user_subscription_updates_id_seq', 1, false);


--
-- Name: user_templates_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.user_templates_id_seq', 1, false);


--
-- Name: user_usage_logs_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.user_usage_logs_id_seq', 1, false);


--
-- Name: users_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.users_id_seq', 1, false);


--
-- Name: wireguard_subnets_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.wireguard_subnets_id_seq', 1, false);


--
-- Name: alembic_version alembic_version_pkc; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.alembic_version
    ADD CONSTRAINT alembic_version_pkc PRIMARY KEY (version_num);


--
-- Name: admin_notification_reminders pk_admin_notification_reminders; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_notification_reminders
    ADD CONSTRAINT pk_admin_notification_reminders PRIMARY KEY (id);


--
-- Name: admin_roles pk_admin_roles; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_roles
    ADD CONSTRAINT pk_admin_roles PRIMARY KEY (id);


--
-- Name: admin_usage_logs pk_admin_usage_logs; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_usage_logs
    ADD CONSTRAINT pk_admin_usage_logs PRIMARY KEY (id);


--
-- Name: admins pk_admins; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admins
    ADD CONSTRAINT pk_admins PRIMARY KEY (id);


--
-- Name: api_keys pk_api_keys; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT pk_api_keys PRIMARY KEY (id);


--
-- Name: client_templates pk_client_templates; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_templates
    ADD CONSTRAINT pk_client_templates PRIMARY KEY (id);


--
-- Name: core_configs pk_core_configs; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.core_configs
    ADD CONSTRAINT pk_core_configs PRIMARY KEY (id);


--
-- Name: groups pk_groups; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.groups
    ADD CONSTRAINT pk_groups PRIMARY KEY (id);


--
-- Name: hosts pk_hosts; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hosts
    ADD CONSTRAINT pk_hosts PRIMARY KEY (id);


--
-- Name: inbounds pk_inbounds; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbounds
    ADD CONSTRAINT pk_inbounds PRIMARY KEY (id);


--
-- Name: inbounds_groups_association pk_inbounds_groups_association; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbounds_groups_association
    ADD CONSTRAINT pk_inbounds_groups_association PRIMARY KEY (inbound_id, group_id);


--
-- Name: jwt pk_jwt; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.jwt
    ADD CONSTRAINT pk_jwt PRIMARY KEY (id);


--
-- Name: next_plans pk_next_plans; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.next_plans
    ADD CONSTRAINT pk_next_plans PRIMARY KEY (id);


--
-- Name: node_stats pk_node_stats; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_stats
    ADD CONSTRAINT pk_node_stats PRIMARY KEY (id);


--
-- Name: node_usage_reset_logs pk_node_usage_reset_logs; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usage_reset_logs
    ADD CONSTRAINT pk_node_usage_reset_logs PRIMARY KEY (id);


--
-- Name: node_usages pk_node_usages; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usages
    ADD CONSTRAINT pk_node_usages PRIMARY KEY (id);


--
-- Name: node_user_usages pk_node_user_usages; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_user_usages
    ADD CONSTRAINT pk_node_user_usages PRIMARY KEY (id);


--
-- Name: nodes pk_nodes; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nodes
    ADD CONSTRAINT pk_nodes PRIMARY KEY (id);


--
-- Name: notification_reminders pk_notification_reminders; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_reminders
    ADD CONSTRAINT pk_notification_reminders PRIMARY KEY (id);


--
-- Name: settings pk_settings; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.settings
    ADD CONSTRAINT pk_settings PRIMARY KEY (id);


--
-- Name: system pk_system; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.system
    ADD CONSTRAINT pk_system PRIMARY KEY (id);


--
-- Name: temp_keys pk_temp_keys; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.temp_keys
    ADD CONSTRAINT pk_temp_keys PRIMARY KEY (key);


--
-- Name: user_hwids pk_user_hwids; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_hwids
    ADD CONSTRAINT pk_user_hwids PRIMARY KEY (id);


--
-- Name: user_subscription_updates pk_user_subscription_updates; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_subscription_updates
    ADD CONSTRAINT pk_user_subscription_updates PRIMARY KEY (id);


--
-- Name: user_templates pk_user_templates; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_templates
    ADD CONSTRAINT pk_user_templates PRIMARY KEY (id);


--
-- Name: user_usage_logs pk_user_usage_logs; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_usage_logs
    ADD CONSTRAINT pk_user_usage_logs PRIMARY KEY (id);


--
-- Name: users pk_users; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT pk_users PRIMARY KEY (id);


--
-- Name: users_groups_association pk_users_groups_association; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users_groups_association
    ADD CONSTRAINT pk_users_groups_association PRIMARY KEY (user_id, groups_id);


--
-- Name: wireguard_subnets pk_wireguard_subnets; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wireguard_subnets
    ADD CONSTRAINT pk_wireguard_subnets PRIMARY KEY (id);


--
-- Name: admin_notification_reminders uq_admin_notification_reminders_admin_id_type_threshold; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_notification_reminders
    ADD CONSTRAINT uq_admin_notification_reminders_admin_id_type_threshold UNIQUE (admin_id, type, threshold);


--
-- Name: admin_roles uq_admin_roles_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_roles
    ADD CONSTRAINT uq_admin_roles_name UNIQUE (name);


--
-- Name: api_keys uq_api_keys_admin_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT uq_api_keys_admin_id UNIQUE (admin_id, name);


--
-- Name: api_keys uq_api_keys_key_hash; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT uq_api_keys_key_hash UNIQUE (key_hash);


--
-- Name: client_templates uq_client_templates_template_type; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.client_templates
    ADD CONSTRAINT uq_client_templates_template_type UNIQUE (template_type, name);


--
-- Name: node_usages uq_node_usages_created_at; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usages
    ADD CONSTRAINT uq_node_usages_created_at UNIQUE (created_at, node_id);


--
-- Name: node_user_usages uq_node_user_usages_created_at; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_user_usages
    ADD CONSTRAINT uq_node_user_usages_created_at UNIQUE (created_at, user_id, node_id);


--
-- Name: nodes uq_nodes_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nodes
    ADD CONSTRAINT uq_nodes_name UNIQUE (name);


--
-- Name: user_hwids uq_user_hwids_user_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_hwids
    ADD CONSTRAINT uq_user_hwids_user_id UNIQUE (user_id, hwid);


--
-- Name: user_templates uq_user_templates_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_templates
    ADD CONSTRAINT uq_user_templates_name UNIQUE (name);


--
-- Name: wireguard_subnets uq_wireguard_subnets_network; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.wireguard_subnets
    ADD CONSTRAINT uq_wireguard_subnets_network UNIQUE (network);


--
-- Name: idx_user_subscription_updates_user_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_subscription_updates_user_created ON public.user_subscription_updates USING btree (user_id, created_at);


--
-- Name: idx_user_subscription_updates_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_user_subscription_updates_user_id ON public.user_subscription_updates USING btree (user_id);


--
-- Name: idx_users_admin_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_admin_created ON public.users USING btree (admin_id, created_at);


--
-- Name: idx_users_admin_online; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_admin_online ON public.users USING btree (admin_id, online_at);


--
-- Name: idx_users_admin_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_admin_status ON public.users USING btree (admin_id, status);


--
-- Name: ix_admin_notification_reminders_admin_id_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_admin_notification_reminders_admin_id_type ON public.admin_notification_reminders USING btree (admin_id, type);


--
-- Name: ix_admins_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ix_admins_username ON public.admins USING btree (username);


--
-- Name: ix_api_keys_admin_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_api_keys_admin_id ON public.api_keys USING btree (admin_id);


--
-- Name: ix_api_keys_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_api_keys_created_at ON public.api_keys USING btree (created_at);


--
-- Name: ix_api_keys_expire_date; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_api_keys_expire_date ON public.api_keys USING btree (expire_date);


--
-- Name: ix_client_templates_template_type; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_client_templates_template_type ON public.client_templates USING btree (template_type);


--
-- Name: ix_inbounds_tag; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ix_inbounds_tag ON public.inbounds USING btree (tag);


--
-- Name: ix_next_plans_user_template_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_next_plans_user_template_id ON public.next_plans USING btree (user_template_id);


--
-- Name: ix_node_usage_reset_logs_node_id_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_node_usage_reset_logs_node_id_created_at ON public.node_usage_reset_logs USING btree (node_id, created_at);


--
-- Name: ix_node_usages_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_node_usages_created_at ON public.node_usages USING btree (created_at);


--
-- Name: ix_node_user_usages_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_node_user_usages_created_at ON public.node_user_usages USING btree (created_at);


--
-- Name: ix_node_user_usages_node_id_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_node_user_usages_node_id_created_at ON public.node_user_usages USING btree (node_id, created_at);


--
-- Name: ix_node_user_usages_user_id_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_node_user_usages_user_id_created_at ON public.node_user_usages USING btree (user_id, created_at);


--
-- Name: ix_user_hwids_created_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_hwids_created_at ON public.user_hwids USING btree (created_at);


--
-- Name: ix_user_hwids_hwid; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_hwids_hwid ON public.user_hwids USING btree (hwid);


--
-- Name: ix_user_hwids_last_used_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_hwids_last_used_at ON public.user_hwids USING btree (last_used_at);


--
-- Name: ix_user_hwids_user_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_hwids_user_id ON public.user_hwids USING btree (user_id);


--
-- Name: ix_user_usage_logs_user_id_reset_at; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX ix_user_usage_logs_user_id_reset_at ON public.user_usage_logs USING btree (user_id, reset_at);


--
-- Name: ix_users_username; Type: INDEX; Schema: public; Owner: -
--

CREATE UNIQUE INDEX ix_users_username ON public.users USING btree (username);


--
-- Name: admin_notification_reminders fk_admin_notification_reminders_admin_id_admins; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_notification_reminders
    ADD CONSTRAINT fk_admin_notification_reminders_admin_id_admins FOREIGN KEY (admin_id) REFERENCES public.admins(id) ON DELETE CASCADE;


--
-- Name: admin_usage_logs fk_admin_usage_logs_admin_id_admins; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admin_usage_logs
    ADD CONSTRAINT fk_admin_usage_logs_admin_id_admins FOREIGN KEY (admin_id) REFERENCES public.admins(id);


--
-- Name: admins fk_admins_role_id_admin_roles; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.admins
    ADD CONSTRAINT fk_admins_role_id_admin_roles FOREIGN KEY (role_id) REFERENCES public.admin_roles(id);


--
-- Name: api_keys fk_api_keys_admin_id_admins; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.api_keys
    ADD CONSTRAINT fk_api_keys_admin_id_admins FOREIGN KEY (admin_id) REFERENCES public.admins(id) ON DELETE CASCADE;


--
-- Name: hosts fk_hosts_inbound_tag_inbounds; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.hosts
    ADD CONSTRAINT fk_hosts_inbound_tag_inbounds FOREIGN KEY (inbound_tag) REFERENCES public.inbounds(tag) ON UPDATE CASCADE ON DELETE SET NULL;


--
-- Name: inbounds_groups_association fk_inbounds_groups_association_group_id_groups; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbounds_groups_association
    ADD CONSTRAINT fk_inbounds_groups_association_group_id_groups FOREIGN KEY (group_id) REFERENCES public.groups(id);


--
-- Name: inbounds_groups_association fk_inbounds_groups_association_inbound_id_inbounds; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.inbounds_groups_association
    ADD CONSTRAINT fk_inbounds_groups_association_inbound_id_inbounds FOREIGN KEY (inbound_id) REFERENCES public.inbounds(id);


--
-- Name: next_plans fk_next_plans_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.next_plans
    ADD CONSTRAINT fk_next_plans_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: next_plans fk_next_plans_user_template_id_user_templates; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.next_plans
    ADD CONSTRAINT fk_next_plans_user_template_id_user_templates FOREIGN KEY (user_template_id) REFERENCES public.user_templates(id) ON DELETE SET NULL;


--
-- Name: node_stats fk_node_stats_node_id_nodes; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_stats
    ADD CONSTRAINT fk_node_stats_node_id_nodes FOREIGN KEY (node_id) REFERENCES public.nodes(id);


--
-- Name: node_usage_reset_logs fk_node_usage_reset_logs_node_id_nodes; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usage_reset_logs
    ADD CONSTRAINT fk_node_usage_reset_logs_node_id_nodes FOREIGN KEY (node_id) REFERENCES public.nodes(id) ON DELETE CASCADE;


--
-- Name: node_usages fk_node_usages_node_id_nodes; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_usages
    ADD CONSTRAINT fk_node_usages_node_id_nodes FOREIGN KEY (node_id) REFERENCES public.nodes(id) ON DELETE CASCADE;


--
-- Name: node_user_usages fk_node_user_usages_node_id_nodes; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_user_usages
    ADD CONSTRAINT fk_node_user_usages_node_id_nodes FOREIGN KEY (node_id) REFERENCES public.nodes(id) ON DELETE CASCADE;


--
-- Name: node_user_usages fk_node_user_usages_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.node_user_usages
    ADD CONSTRAINT fk_node_user_usages_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: notification_reminders fk_notification_reminders_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.notification_reminders
    ADD CONSTRAINT fk_notification_reminders_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: template_group_association fk_template_group_association_group_id_groups; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.template_group_association
    ADD CONSTRAINT fk_template_group_association_group_id_groups FOREIGN KEY (group_id) REFERENCES public.groups(id);


--
-- Name: template_group_association fk_template_group_association_user_template_id_user_templates; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.template_group_association
    ADD CONSTRAINT fk_template_group_association_user_template_id_user_templates FOREIGN KEY (user_template_id) REFERENCES public.user_templates(id);


--
-- Name: user_hwids fk_user_hwids_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_hwids
    ADD CONSTRAINT fk_user_hwids_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_subscription_updates fk_user_subscription_updates_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_subscription_updates
    ADD CONSTRAINT fk_user_subscription_updates_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: user_usage_logs fk_user_usage_logs_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.user_usage_logs
    ADD CONSTRAINT fk_user_usage_logs_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: users fk_users_admin_id_admins; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT fk_users_admin_id_admins FOREIGN KEY (admin_id) REFERENCES public.admins(id);


--
-- Name: users_groups_association fk_users_groups_association_groups_id_groups; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users_groups_association
    ADD CONSTRAINT fk_users_groups_association_groups_id_groups FOREIGN KEY (groups_id) REFERENCES public.groups(id);


--
-- Name: users_groups_association fk_users_groups_association_user_id_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users_groups_association
    ADD CONSTRAINT fk_users_groups_association_user_id_users FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- Name: nodes nodes_ibfk_1; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.nodes
    ADD CONSTRAINT nodes_ibfk_1 FOREIGN KEY (core_config_id) REFERENCES public.core_configs(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--



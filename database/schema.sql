--
-- PostgreSQL database dump
--

\restrict KzGYvSeUFlvbmGTFubRCy0wus2Fk0vzURIDHWDVrTN0ot4GbLwIDoqZIXBE6Vye

-- Dumped from database version 18.3
-- Dumped by pg_dump version 18.3

-- Started on 2026-04-22 13:27:18

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
-- TOC entry 5 (class 2615 OID 2200)
-- Name: public; Type: SCHEMA; Schema: -; Owner: pg_database_owner
--

CREATE SCHEMA public;


ALTER SCHEMA public OWNER TO pg_database_owner;

--
-- TOC entry 5117 (class 0 OID 0)
-- Dependencies: 5
-- Name: SCHEMA public; Type: COMMENT; Schema: -; Owner: pg_database_owner
--

COMMENT ON SCHEMA public IS 'standard public schema';


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 225 (class 1259 OID 16499)
-- Name: barrier_logs; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.barrier_logs (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid,
    access_type character varying(20),
    direction character varying(20),
    car_number character varying(50),
    notes text,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.barrier_logs OWNER TO postgres;

--
-- TOC entry 224 (class 1259 OID 16479)
-- Name: guest_access; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.guest_access (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    resident_id uuid NOT NULL,
    guest_name character varying(255) NOT NULL,
    guest_phone character varying(50),
    car_number character varying(50),
    access_type character varying(20) DEFAULT 'walk'::character varying,
    access_code character varying(20) NOT NULL,
    valid_from timestamp with time zone NOT NULL,
    valid_until timestamp with time zone NOT NULL,
    status character varying(20) DEFAULT 'active'::character varying,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.guest_access OWNER TO postgres;

--
-- TOC entry 221 (class 1259 OID 16415)
-- Name: profiles; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.profiles (
    id uuid NOT NULL,
    full_name character varying(255) DEFAULT ''::character varying,
    email character varying(255) DEFAULT ''::character varying,
    phone character varying(50) DEFAULT ''::character varying,
    iin character varying(20) DEFAULT ''::character varying,
    person_type character varying(50) DEFAULT ''::character varying,
    city character varying(100) DEFAULT ''::character varying,
    street character varying(255) DEFAULT ''::character varying,
    property_type character varying(50) DEFAULT ''::character varying,
    property_number character varying(50) DEFAULT ''::character varying,
    full_address text DEFAULT ''::text,
    role character varying(50) DEFAULT 'resident'::character varying,
    verification_status character varying(50) DEFAULT 'not_submitted'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.profiles OWNER TO postgres;

--
-- TOC entry 223 (class 1259 OID 16462)
-- Name: request_photos; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.request_photos (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    request_id uuid NOT NULL,
    file_path character varying(500) NOT NULL,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.request_photos OWNER TO postgres;

--
-- TOC entry 222 (class 1259 OID 16442)
-- Name: service_requests; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.service_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    category character varying(100) NOT NULL,
    description text NOT NULL,
    status character varying(50) DEFAULT 'new'::character varying,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.service_requests OWNER TO postgres;

--
-- TOC entry 220 (class 1259 OID 16400)
-- Name: users; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.users (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    email character varying(255) NOT NULL,
    password_hash character varying(255) NOT NULL,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.users OWNER TO postgres;

--
-- TOC entry 227 (class 1259 OID 16539)
-- Name: verification_documents; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.verification_documents (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    verification_request_id uuid NOT NULL,
    file_path character varying(500) NOT NULL,
    file_name character varying(255) NOT NULL,
    file_size bigint DEFAULT 0,
    created_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.verification_documents OWNER TO postgres;

--
-- TOC entry 226 (class 1259 OID 16514)
-- Name: verification_requests; Type: TABLE; Schema: public; Owner: postgres
--

CREATE TABLE public.verification_requests (
    id uuid DEFAULT public.uuid_generate_v4() NOT NULL,
    user_id uuid NOT NULL,
    requested_role character varying(50) NOT NULL,
    comment text DEFAULT ''::text,
    status character varying(50) DEFAULT 'pending'::character varying,
    reviewed_by uuid,
    reviewed_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now(),
    updated_at timestamp with time zone DEFAULT now()
);


ALTER TABLE public.verification_requests OWNER TO postgres;

--
-- TOC entry 5109 (class 0 OID 16499)
-- Dependencies: 225
-- Data for Name: barrier_logs; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.barrier_logs (id, user_id, access_type, direction, car_number, notes, created_at) FROM stdin;
3b7e6122-79be-4f81-8094-2d7e22b5653a	edb2ffde-ef40-4561-9c79-5fd58b97d332	car	in	\N	Открытие из приложения	2026-04-21 20:11:38.275536+05
\.


--
-- TOC entry 5108 (class 0 OID 16479)
-- Dependencies: 224
-- Data for Name: guest_access; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.guest_access (id, resident_id, guest_name, guest_phone, car_number, access_type, access_code, valid_from, valid_until, status, created_at) FROM stdin;
1f399e95-235c-4c29-9e1d-4fff2613c43f	edb2ffde-ef40-4561-9c79-5fd58b97d332	nurs	\N	872ajjj14	car	G-953969	2026-04-21 19:32:00+05	2026-04-30 19:32:00+05	cancelled	2026-04-21 19:32:33.96951+05
\.


--
-- TOC entry 5105 (class 0 OID 16415)
-- Dependencies: 221
-- Data for Name: profiles; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.profiles (id, full_name, email, phone, iin, person_type, city, street, property_type, property_number, full_address, role, verification_status, created_at, updated_at) FROM stdin;
78a8b641-e330-429f-82e8-770976250a64	����� ����	admin@test.com	+7 777 000 0000	123456789012	owner	������	���� 1	apartment	42	������, ���� 1, �� 42	resident	not_submitted	2026-04-21 19:00:34.990064+05	2026-04-21 19:00:34.990064+05
edb2ffde-ef40-4561-9c79-5fd58b97d332	оразы	test@gmail.com	87027156005	123456789012	individual					4GG6+7JF, Astana, Astana	admin	approved	2026-04-21 19:30:56.157107+05	2026-04-21 20:29:26.137571+05
\.


--
-- TOC entry 5107 (class 0 OID 16462)
-- Dependencies: 223
-- Data for Name: request_photos; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.request_photos (id, request_id, file_path, created_at) FROM stdin;
b294974c-5ef4-42bf-97d8-394ad58b5e9b	9e620344-09ff-4298-b153-89718cc3fd6d	uploads\\request-photos\\5688c6df-e34e-436a-a609-ccc338f103a7.jpg	2026-04-21 19:31:56.509495+05
\.


--
-- TOC entry 5106 (class 0 OID 16442)
-- Dependencies: 222
-- Data for Name: service_requests; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.service_requests (id, user_id, category, description, status, created_at, updated_at) FROM stdin;
9e620344-09ff-4298-b153-89718cc3fd6d	edb2ffde-ef40-4561-9c79-5fd58b97d332	Лифт	123456test	done	2026-04-21 19:31:55.91585+05	2026-04-21 20:12:47.205801+05
\.


--
-- TOC entry 5104 (class 0 OID 16400)
-- Dependencies: 220
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.users (id, email, password_hash, created_at, updated_at) FROM stdin;
78a8b641-e330-429f-82e8-770976250a64	admin@test.com	$2a$10$dVFRKS/.N0gDA3H6lTbd7eMewXj0CYFEQAabHaMYCdBOS.zOYpN7.	2026-04-21 19:00:34.986199+05	2026-04-21 19:00:34.986199+05
edb2ffde-ef40-4561-9c79-5fd58b97d332	test@gmail.com	$2a$10$3so/IqHtcxG/IZDJlwjs0OeR.onLCFWNVAfRcku/pZkTUnlgnqsPa	2026-04-21 19:30:56.151243+05	2026-04-21 19:30:56.151243+05
\.


--
-- TOC entry 5111 (class 0 OID 16539)
-- Dependencies: 227
-- Data for Name: verification_documents; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.verification_documents (id, verification_request_id, file_path, file_name, file_size, created_at) FROM stdin;
d14091bf-c6a4-4a60-865c-87896f0981e8	3165f27d-58ec-4884-ba82-470729c37a66	uploads\\verification-docs\\edb2ffde-ef40-4561-9c79-5fd58b97d332\\1776785105253_090ef0ed.png	Screenshot_20260421-071246.png	25535	2026-04-21 20:25:05.255313+05
\.


--
-- TOC entry 5110 (class 0 OID 16514)
-- Dependencies: 226
-- Data for Name: verification_requests; Type: TABLE DATA; Schema: public; Owner: postgres
--

COPY public.verification_requests (id, user_id, requested_role, comment, status, reviewed_by, reviewed_at, created_at, updated_at) FROM stdin;
3165f27d-58ec-4884-ba82-470729c37a66	edb2ffde-ef40-4561-9c79-5fd58b97d332	owner	test1234	approved	edb2ffde-ef40-4561-9c79-5fd58b97d332	2026-04-21 20:29:26.136464+05	2026-04-21 20:25:04.710894+05	2026-04-21 20:29:26.136464+05
\.


--
-- TOC entry 4944 (class 2606 OID 16508)
-- Name: barrier_logs barrier_logs_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.barrier_logs
    ADD CONSTRAINT barrier_logs_pkey PRIMARY KEY (id);


--
-- TOC entry 4942 (class 2606 OID 16493)
-- Name: guest_access guest_access_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guest_access
    ADD CONSTRAINT guest_access_pkey PRIMARY KEY (id);


--
-- TOC entry 4936 (class 2606 OID 16436)
-- Name: profiles profiles_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_pkey PRIMARY KEY (id);


--
-- TOC entry 4940 (class 2606 OID 16473)
-- Name: request_photos request_photos_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.request_photos
    ADD CONSTRAINT request_photos_pkey PRIMARY KEY (id);


--
-- TOC entry 4938 (class 2606 OID 16456)
-- Name: service_requests service_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.service_requests
    ADD CONSTRAINT service_requests_pkey PRIMARY KEY (id);


--
-- TOC entry 4932 (class 2606 OID 16414)
-- Name: users users_email_key; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_email_key UNIQUE (email);


--
-- TOC entry 4934 (class 2606 OID 16412)
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- TOC entry 4948 (class 2606 OID 16552)
-- Name: verification_documents verification_documents_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.verification_documents
    ADD CONSTRAINT verification_documents_pkey PRIMARY KEY (id);


--
-- TOC entry 4946 (class 2606 OID 16528)
-- Name: verification_requests verification_requests_pkey; Type: CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.verification_requests
    ADD CONSTRAINT verification_requests_pkey PRIMARY KEY (id);


--
-- TOC entry 4953 (class 2606 OID 16509)
-- Name: barrier_logs barrier_logs_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.barrier_logs
    ADD CONSTRAINT barrier_logs_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id);


--
-- TOC entry 4952 (class 2606 OID 16494)
-- Name: guest_access guest_access_resident_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.guest_access
    ADD CONSTRAINT guest_access_resident_id_fkey FOREIGN KEY (resident_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- TOC entry 4949 (class 2606 OID 16437)
-- Name: profiles profiles_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.profiles
    ADD CONSTRAINT profiles_id_fkey FOREIGN KEY (id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- TOC entry 4951 (class 2606 OID 16474)
-- Name: request_photos request_photos_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.request_photos
    ADD CONSTRAINT request_photos_request_id_fkey FOREIGN KEY (request_id) REFERENCES public.service_requests(id) ON DELETE CASCADE;


--
-- TOC entry 4950 (class 2606 OID 16457)
-- Name: service_requests service_requests_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.service_requests
    ADD CONSTRAINT service_requests_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- TOC entry 4956 (class 2606 OID 16553)
-- Name: verification_documents verification_documents_verification_request_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.verification_documents
    ADD CONSTRAINT verification_documents_verification_request_id_fkey FOREIGN KEY (verification_request_id) REFERENCES public.verification_requests(id) ON DELETE CASCADE;


--
-- TOC entry 4954 (class 2606 OID 16534)
-- Name: verification_requests verification_requests_reviewed_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.verification_requests
    ADD CONSTRAINT verification_requests_reviewed_by_fkey FOREIGN KEY (reviewed_by) REFERENCES public.users(id);


--
-- TOC entry 4955 (class 2606 OID 16529)
-- Name: verification_requests verification_requests_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: postgres
--

ALTER TABLE ONLY public.verification_requests
    ADD CONSTRAINT verification_requests_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


-- Completed on 2026-04-22 13:27:19

--
-- PostgreSQL database dump complete
--

\unrestrict KzGYvSeUFlvbmGTFubRCy0wus2Fk0vzURIDHWDVrTN0ot4GbLwIDoqZIXBE6Vye


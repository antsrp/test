CREATE TABLE public.users
(
    id bigint NOT NULL PRIMARY KEY,
    balance bigint NOT NULL
);

CREATE TABLE IF NOT EXISTS public.favors
(
    id bigint NOT NULL,
    name character varying(100) NOT NULL,
    CONSTRAINT favors_pkey PRIMARY KEY (id)
);

CREATE TABLE IF NOT EXISTS public.chains(
    id SERIAL PRIMARY KEY,
    order_id bigint NOT NULL,
    service_id bigint NOT NULL
);

CREATE TABLE IF NOT EXISTS public.transactions
(
    id SERIAL PRIMARY KEY,
    user_id bigint NOT NULL,
    direction character varying(10) NOT NULL,
    closed_at timestamp with time zone,
    chain_id bigint,
    is_completed boolean NOT NULL,
    cost bigint NOT NULL,
    comment character varying(50)
);

INSERT INTO favors (id, name) VALUES 
(1, 'Favor 1'),
(2, 'Favor 2'),
(3, 'Favor 3'),
(4, 'Favor 4'),
(5, 'Favor 5'),
(6, 'Favor 6'),
(7, 'Favor 7'),
(8, 'Favor 8');
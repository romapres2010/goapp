-- Create schemas section -------------------------------------------------

CREATE SCHEMA IF NOT EXISTS app
;

-- Create tables section -------------------------------------------------

-- Table app.currencies

CREATE TABLE IF NOT EXISTS app.currencies
(
    currency_id Integer NOT NULL GENERATED ALWAYS AS IDENTITY
(INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1),
    currency_code Character varying(256) NOT NULL,
    currency_name Character varying(2000) NOT NULL,
    currency_name_alt Character varying(2000),
    description Character varying(2000),
    enabled_flag Character varying(1) DEFAULT 'Y' NOT NULL
    CONSTRAINT chk_enabled_flag CHECK (enabled_flag in ('Y', 'N')),
    creation_date Timestamp DEFAULT Now()  NOT NULL,
    created_by Character varying(256) DEFAULT 'none' NOT NULL,
    last_update_date Timestamp DEFAULT Now() NOT NULL,
    last_updated_by Character varying(256) DEFAULT 'none' NOT NULL
    )
    WITH (
        autovacuum_enabled=true)
;
COMMENT ON TABLE app.currencies IS 'Валюты; SHORT_NAME=cur; SINGLE_NAME=currency;'
;
COMMENT ON COLUMN app.currencies.currency_id IS 'Идентификатор валюты'
;
COMMENT ON COLUMN app.currencies.currency_code IS 'Код валюты'
;
COMMENT ON COLUMN app.currencies.currency_name IS 'Наименование валюты'
;
COMMENT ON COLUMN app.currencies.currency_name_alt IS 'Наименование валюты альтернативное'
;
COMMENT ON COLUMN app.currencies.description IS 'Примечание'
;
COMMENT ON COLUMN app.currencies.enabled_flag IS 'Признак активной записи'
;
COMMENT ON COLUMN app.currencies.creation_date IS 'Дата создания'
;
COMMENT ON COLUMN app.currencies.created_by IS 'Пользователь, создавший запись'
;
COMMENT ON COLUMN app.currencies.last_update_date IS 'Дата последнего изменения'
;
COMMENT ON COLUMN app.currencies.last_updated_by IS 'Пользователь, внесший последние изменения'
;

ALTER TABLE app.currencies ADD CONSTRAINT currencies_pk PRIMARY KEY (currency_id)
;

ALTER TABLE app.currencies ADD CONSTRAINT currencies_uk1 UNIQUE (currency_code)
;

ALTER TABLE app.currencies ADD CONSTRAINT currencies_uk2 UNIQUE (currency_name)
;

-- Table app.countries

CREATE TABLE IF NOT EXISTS app.countries
(
    country_id Integer NOT NULL GENERATED ALWAYS AS IDENTITY
(INCREMENT BY 1 NO MINVALUE NO MAXVALUE CACHE 1),
    country_code Character varying(256) NOT NULL,
    country_name Character varying(2000) NOT NULL,
    country_name_alt Character varying(2000),
    description Character varying(2000),
    enabled_flag Character varying(1) DEFAULT 'Y' NOT NULL
    CONSTRAINT chk_enabled_flag CHECK (enabled_flag in ('Y', 'N')),
    creation_date Timestamp DEFAULT Now()  NOT NULL,
    created_by Character varying(256) DEFAULT 'none' NOT NULL,
    last_update_date Timestamp DEFAULT Now() NOT NULL,
    last_updated_by Character varying(256) DEFAULT 'none' NOT NULL
    )
    WITH (
        autovacuum_enabled=true)
;
COMMENT ON TABLE app.countries IS 'Страны; SHORT_NAME=cntr; SINGLE_NAME=country;'
;
COMMENT ON COLUMN app.countries.country_id IS 'Идентификатор региона'
;
COMMENT ON COLUMN app.countries.country_code IS 'Код региона'
;
COMMENT ON COLUMN app.countries.country_name IS 'Наименование региона'
;
COMMENT ON COLUMN app.countries.country_name_alt IS 'Альтернативное наименование региона'
;
COMMENT ON COLUMN app.countries.description IS 'Примечание'
;
COMMENT ON COLUMN app.countries.enabled_flag IS 'Признак активной записи'
;
COMMENT ON COLUMN app.countries.creation_date IS 'Дата создания'
;
COMMENT ON COLUMN app.countries.created_by IS 'Пользователь, создавший запись'
;
COMMENT ON COLUMN app.countries.last_update_date IS 'Дата последнего изменения'
;
COMMENT ON COLUMN app.countries.last_updated_by IS 'Пользователь, внесший последние изменения'
;

ALTER TABLE app.countries ADD CONSTRAINT countries_pk PRIMARY KEY (country_id)
;

ALTER TABLE app.countries ADD CONSTRAINT countries_uk1 UNIQUE (country_code)
;

ALTER TABLE app.countries ADD CONSTRAINT countries_uk2 UNIQUE (country_name)
;

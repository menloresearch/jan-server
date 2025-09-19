-- Create "api_key" table
CREATE TABLE "public"."api_key" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "public_id" character varying(128) NOT NULL,
  "key_hash" character varying(128) NOT NULL,
  "plaintext_hint" character varying(16) NULL,
  "description" character varying(255) NULL,
  "enabled" boolean NULL DEFAULT true,
  "apikey_type" character varying(32) NOT NULL,
  "owner_public_id" character varying(50) NOT NULL,
  "organization_id" bigint NULL,
  "project_id" bigint NULL,
  "permissions" json NULL,
  "expires_at" timestamp NULL,
  "last_used_at" timestamp NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_api_key_apikey_type" to table: "api_key"
CREATE INDEX "idx_api_key_apikey_type" ON "public"."api_key" ("apikey_type");
-- Create index "idx_api_key_deleted_at" to table: "api_key"
CREATE INDEX "idx_api_key_deleted_at" ON "public"."api_key" ("deleted_at");
-- Create index "idx_api_key_enabled" to table: "api_key"
CREATE INDEX "idx_api_key_enabled" ON "public"."api_key" ("enabled");
-- Create index "idx_api_key_key_hash" to table: "api_key"
CREATE UNIQUE INDEX "idx_api_key_key_hash" ON "public"."api_key" ("key_hash");
-- Create index "idx_api_key_organization_id" to table: "api_key"
CREATE INDEX "idx_api_key_organization_id" ON "public"."api_key" ("organization_id");
-- Create index "idx_api_key_owner_public_id" to table: "api_key"
CREATE UNIQUE INDEX "idx_api_key_owner_public_id" ON "public"."api_key" ("owner_public_id");
-- Create index "idx_api_key_project_id" to table: "api_key"
CREATE INDEX "idx_api_key_project_id" ON "public"."api_key" ("project_id");
-- Create index "idx_api_key_public_id" to table: "api_key"
CREATE UNIQUE INDEX "idx_api_key_public_id" ON "public"."api_key" ("public_id");
-- Create "user" table
CREATE TABLE "public"."user" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(100) NOT NULL,
  "email" character varying(255) NOT NULL,
  "public_id" character varying(50) NOT NULL,
  "enabled" boolean NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_user_deleted_at" to table: "user"
CREATE INDEX "idx_user_deleted_at" ON "public"."user" ("deleted_at");
-- Create index "idx_user_email" to table: "user"
CREATE UNIQUE INDEX "idx_user_email" ON "public"."user" ("email");
-- Create index "idx_user_public_id" to table: "user"
CREATE UNIQUE INDEX "idx_user_public_id" ON "public"."user" ("public_id");
-- Create "conversation" table
CREATE TABLE "public"."conversation" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "public_id" character varying(50) NOT NULL,
  "title" character varying(255) NULL,
  "user_id" bigint NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "metadata" text NULL,
  "is_private" boolean NOT NULL DEFAULT true,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_conversation_user" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_conversation_deleted_at" to table: "conversation"
CREATE INDEX "idx_conversation_deleted_at" ON "public"."conversation" ("deleted_at");
-- Create index "idx_conversation_public_id" to table: "conversation"
CREATE UNIQUE INDEX "idx_conversation_public_id" ON "public"."conversation" ("public_id");
-- Create index "idx_conversation_user_id" to table: "conversation"
CREATE INDEX "idx_conversation_user_id" ON "public"."conversation" ("user_id");
-- Create "item" table
CREATE TABLE "public"."item" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "public_id" character varying(50) NOT NULL,
  "conversation_id" bigint NOT NULL,
  "type" character varying(50) NOT NULL,
  "role" character varying(20) NULL,
  "content" text NULL,
  "status" character varying(50) NULL,
  "incomplete_at" bigint NULL,
  "incomplete_details" text NULL,
  "completed_at" bigint NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_conversation_items" FOREIGN KEY ("conversation_id") REFERENCES "public"."conversation" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_item_conversation_id" to table: "item"
CREATE INDEX "idx_item_conversation_id" ON "public"."item" ("conversation_id");
-- Create index "idx_item_deleted_at" to table: "item"
CREATE INDEX "idx_item_deleted_at" ON "public"."item" ("deleted_at");
-- Create index "idx_item_public_id" to table: "item"
CREATE UNIQUE INDEX "idx_item_public_id" ON "public"."item" ("public_id");
-- Create "organization" table
CREATE TABLE "public"."organization" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(128) NOT NULL,
  "public_id" character varying(64) NOT NULL,
  "enabled" boolean NULL DEFAULT true,
  "owner_id" bigint NOT NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_organization_owner" FOREIGN KEY ("owner_id") REFERENCES "public"."user" ("id") ON UPDATE CASCADE ON DELETE SET NULL
);
-- Create index "idx_organization_deleted_at" to table: "organization"
CREATE INDEX "idx_organization_deleted_at" ON "public"."organization" ("deleted_at");
-- Create index "idx_organization_enabled" to table: "organization"
CREATE INDEX "idx_organization_enabled" ON "public"."organization" ("enabled");
-- Create index "idx_organization_name" to table: "organization"
CREATE UNIQUE INDEX "idx_organization_name" ON "public"."organization" ("name");
-- Create index "idx_organization_owner_id" to table: "organization"
CREATE INDEX "idx_organization_owner_id" ON "public"."organization" ("owner_id");
-- Create index "idx_organization_public_id" to table: "organization"
CREATE UNIQUE INDEX "idx_organization_public_id" ON "public"."organization" ("public_id");
-- Create "organization_member" table
CREATE TABLE "public"."organization_member" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "user_id" bigint NOT NULL,
  "organization_id" bigint NOT NULL,
  "role" character varying(20) NOT NULL,
  PRIMARY KEY ("id", "user_id", "organization_id"),
  CONSTRAINT "fk_organization_members" FOREIGN KEY ("organization_id") REFERENCES "public"."organization" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_user_organizations" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_organization_member_deleted_at" to table: "organization_member"
CREATE INDEX "idx_organization_member_deleted_at" ON "public"."organization_member" ("deleted_at");
-- Create "project" table
CREATE TABLE "public"."project" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "name" character varying(128) NOT NULL,
  "public_id" character varying(50) NOT NULL,
  "status" character varying(20) NOT NULL DEFAULT 'active',
  "organization_id" bigint NOT NULL,
  "archived_at" timestamptz NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_project_archived_at" to table: "project"
CREATE INDEX "idx_project_archived_at" ON "public"."project" ("archived_at");
-- Create index "idx_project_deleted_at" to table: "project"
CREATE INDEX "idx_project_deleted_at" ON "public"."project" ("deleted_at");
-- Create index "idx_project_name" to table: "project"
CREATE UNIQUE INDEX "idx_project_name" ON "public"."project" ("name");
-- Create index "idx_project_organization_id" to table: "project"
CREATE INDEX "idx_project_organization_id" ON "public"."project" ("organization_id");
-- Create index "idx_project_public_id" to table: "project"
CREATE UNIQUE INDEX "idx_project_public_id" ON "public"."project" ("public_id");
-- Create index "idx_project_status" to table: "project"
CREATE INDEX "idx_project_status" ON "public"."project" ("status");
-- Create "project_member" table
CREATE TABLE "public"."project_member" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "user_id" bigint NOT NULL,
  "project_id" bigint NOT NULL,
  "role" character varying(20) NOT NULL,
  PRIMARY KEY ("id", "user_id", "project_id"),
  CONSTRAINT "fk_project_members" FOREIGN KEY ("project_id") REFERENCES "public"."project" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_user_projects" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_project_member_deleted_at" to table: "project_member"
CREATE INDEX "idx_project_member_deleted_at" ON "public"."project_member" ("deleted_at");

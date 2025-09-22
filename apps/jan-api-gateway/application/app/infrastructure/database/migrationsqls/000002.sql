-- Drop index "idx_api_key_owner_public_id" from table: "api_key"
DROP INDEX "public"."idx_api_key_owner_public_id";
-- Create index "idx_conversation_is_private" to table: "conversation"
CREATE INDEX "idx_conversation_is_private" ON "public"."conversation" ("is_private");
-- Create index "idx_conversation_status" to table: "conversation"
CREATE INDEX "idx_conversation_status" ON "public"."conversation" ("status");
-- Drop index "idx_organization_name" from table: "organization"
DROP INDEX "public"."idx_organization_name";
-- Modify "organization_member" table
ALTER TABLE "public"."organization_member" ADD COLUMN "is_primary" boolean NULL DEFAULT false;
-- Drop index "idx_project_name" from table: "project"
DROP INDEX "public"."idx_project_name";
-- Create "invite" table
CREATE TABLE "public"."invite" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "public_id" character varying(64) NOT NULL,
  "email" character varying(128) NOT NULL,
  "role" character varying(20) NOT NULL,
  "status" character varying(20) NOT NULL,
  "invited_at" timestamptz NULL,
  "expires_at" timestamptz NULL,
  "accepted_at" timestamptz NULL,
  "secrets" text NULL,
  "projects" jsonb NULL,
  "organization_id" bigint NOT NULL,
  PRIMARY KEY ("id")
);
-- Create index "idx_invite_deleted_at" to table: "invite"
CREATE INDEX "idx_invite_deleted_at" ON "public"."invite" ("deleted_at");
-- Create index "idx_invite_organization_id" to table: "invite"
CREATE INDEX "idx_invite_organization_id" ON "public"."invite" ("organization_id");
-- Create index "idx_invite_public_id" to table: "invite"
CREATE UNIQUE INDEX "idx_invite_public_id" ON "public"."invite" ("public_id");
-- Create index "idx_invite_status" to table: "invite"
CREATE INDEX "idx_invite_status" ON "public"."invite" ("status");
-- Modify "user" table
ALTER TABLE "public"."user" ADD COLUMN "is_guest" boolean NULL;
-- Create "responses" table
CREATE TABLE "public"."responses" (
  "id" bigserial NOT NULL,
  "created_at" timestamptz NULL,
  "updated_at" timestamptz NULL,
  "deleted_at" timestamptz NULL,
  "public_id" character varying(255) NOT NULL,
  "user_id" bigint NOT NULL,
  "conversation_id" bigint NULL,
  "previous_response_id" character varying(255) NULL,
  "model" character varying(255) NOT NULL,
  "status" character varying(50) NOT NULL DEFAULT 'pending',
  "input" text NOT NULL,
  "output" text NULL,
  "system_prompt" text NULL,
  "max_tokens" bigint NULL,
  "temperature" numeric NULL,
  "top_p" numeric NULL,
  "top_k" bigint NULL,
  "repetition_penalty" numeric NULL,
  "seed" bigint NULL,
  "stop" text NULL,
  "presence_penalty" numeric NULL,
  "frequency_penalty" numeric NULL,
  "logit_bias" text NULL,
  "response_format" text NULL,
  "tools" text NULL,
  "tool_choice" text NULL,
  "metadata" text NULL,
  "stream" boolean NULL,
  "background" boolean NULL,
  "timeout" bigint NULL,
  "user" character varying(255) NULL,
  "usage" text NULL,
  "error" text NULL,
  "completed_at" timestamptz NULL,
  "cancelled_at" timestamptz NULL,
  "failed_at" timestamptz NULL,
  PRIMARY KEY ("id"),
  CONSTRAINT "fk_responses_conversation" FOREIGN KEY ("conversation_id") REFERENCES "public"."conversation" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION,
  CONSTRAINT "fk_responses_user_entity" FOREIGN KEY ("user_id") REFERENCES "public"."user" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION
);
-- Create index "idx_responses_conversation_id" to table: "responses"
CREATE INDEX "idx_responses_conversation_id" ON "public"."responses" ("conversation_id");
-- Create index "idx_responses_deleted_at" to table: "responses"
CREATE INDEX "idx_responses_deleted_at" ON "public"."responses" ("deleted_at");
-- Create index "idx_responses_model" to table: "responses"
CREATE INDEX "idx_responses_model" ON "public"."responses" ("model");
-- Create index "idx_responses_previous_response_id" to table: "responses"
CREATE INDEX "idx_responses_previous_response_id" ON "public"."responses" ("previous_response_id");
-- Create index "idx_responses_public_id" to table: "responses"
CREATE UNIQUE INDEX "idx_responses_public_id" ON "public"."responses" ("public_id");
-- Create index "idx_responses_status" to table: "responses"
CREATE INDEX "idx_responses_status" ON "public"."responses" ("status");
-- Create index "idx_responses_user_id" to table: "responses"
CREATE INDEX "idx_responses_user_id" ON "public"."responses" ("user_id");
-- Modify "item" table
ALTER TABLE "public"."item" ALTER COLUMN "incomplete_at" TYPE timestamp, ALTER COLUMN "completed_at" TYPE timestamp, ADD COLUMN "response_id" bigint NULL, ADD CONSTRAINT "fk_responses_items" FOREIGN KEY ("response_id") REFERENCES "public"."responses" ("id") ON UPDATE NO ACTION ON DELETE NO ACTION;
-- Create index "idx_item_response_id" to table: "item"
CREATE INDEX "idx_item_response_id" ON "public"."item" ("response_id");
-- Create index "idx_item_role" to table: "item"
CREATE INDEX "idx_item_role" ON "public"."item" ("role");
-- Create index "idx_item_status" to table: "item"
CREATE INDEX "idx_item_status" ON "public"."item" ("status");
-- Create index "idx_item_type" to table: "item"
CREATE INDEX "idx_item_type" ON "public"."item" ("type");

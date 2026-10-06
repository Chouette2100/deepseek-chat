ALTER TABLE qa_records
  ADD COLUMN attachmentname varchar(255) NOT NULL DEFAULT '',
  ADD COLUMN attachmentmime varchar(128) NOT NULL DEFAULT '',
  ADD COLUMN attachmentdata longtext NOT NULL DEFAULT '';

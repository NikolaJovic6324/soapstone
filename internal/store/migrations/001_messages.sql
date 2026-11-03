-- Tabela za poruke. id dodeljuje baza i samo raste, kao u memorijskom
-- skladištu; created_at je bez delova sekunde, takođe kao tamo.
CREATE TABLE messages (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    author     text        NOT NULL,
    text       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT date_trunc('second', now())
);

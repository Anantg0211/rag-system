# rag-document-assistant

A learning-focused Retrieval-Augmented Generation (RAG) backend built with Go,
Gin, PostgreSQL, Qdrant, and OpenAI. The project intentionally implements the
RAG pipeline directly—without LangChain or LlamaIndex—so every step remains
visible, testable, and explainable.

> Current status: architecture and implementation plan only. Phase 1 has not
> started yet.

## What this application will do

1. Accept a text-based PDF upload.
2. Extract text page by page.
3. Split each page into slightly overlapping chunks.
4. Generate one embedding vector for each chunk.
5. Store durable document/chunk records in PostgreSQL and searchable vectors in
   Qdrant.
6. Embed a user's question with the same embedding model.
7. Search Qdrant for the closest chunk vectors.
8. Apply the configured Top-K limit and minimum similarity score.
9. Give the surviving chunks and the question to the answer model.
10. Return a grounded answer with filename/page/chunk citations, or the exact
    fallback: `I don't have enough information to answer that.`

## RAG in simple terms

An LLM does not automatically know the contents of an uploaded PDF. RAG gives
it a small, relevant portion of those contents at question time.

An **embedding** is a list of numbers representing the meaning of text. Texts
with similar meaning generally produce vectors that are close together. The
embedding is not a summary and cannot be read by a human; it is a search key.

Qdrant is a **vector database**. It stores each chunk's vector alongside a
payload containing human-readable metadata. Given a question vector, Qdrant
finds the closest chunk vectors using a similarity metric (cosine similarity in
this project).

The full flow has two paths:

```text
INGESTION (done when a PDF is uploaded)

PDF -> page text -> overlapping chunks -> chunk embeddings -> Qdrant
             |              |                                  |
             +--------------+----------> PostgreSQL metadata <-+

QUERY (done for every question)

question -> question embedding -> Qdrant similarity search
                                      |
                                      v
                              top matching chunks
                                      |
                                      v
                     grounded prompt (context + question)
                                      |
                                      v
                            LLM answer + citations
```

The retrieval stage reduces a potentially large document collection to a few
relevant passages. The generation stage is then explicitly instructed to use
only those passages. Retrieval provides evidence; generation turns that
evidence into a readable answer.

## Architecture proposal

The sibling `../ai-health-bot` project uses a straightforward
router/controller/service/model/database layout. This project keeps that
familiar request flow while making boundaries explicit and dependencies
replaceable:

```text
Gin route
  -> controller (validation and response mapping)
  -> application service (use-case orchestration)
  -> interfaces (ports)
       -> PostgreSQL repository
       -> PDF extractor
       -> chunker
       -> OpenAI embedding provider
       -> Qdrant vector store
       -> OpenAI answer generator
```

### Component responsibilities

| Component | Responsibility | Must not do |
| --- | --- | --- |
| Controller | Parse requests, validate input, map service errors to HTTP responses | Contain RAG logic or call vendors directly |
| Document service | Coordinate upload, extraction, chunking, persistence, embedding, and indexing | Know Gin request/response types |
| Query service | Embed a question, retrieve chunks, build grounded context, request an answer | Know Qdrant/OpenAI concrete client details |
| PostgreSQL repository | Persist documents, chunks, statuses, and indexing information | Perform semantic search |
| Embedding provider | Convert text batches into vectors | Store vectors or generate answers |
| Vector store | Create/check the collection, upsert points, and perform similarity search | Generate embeddings |
| Answer generator | Produce an answer from supplied context and question | Retrieve documents itself |
| PDF extractor | Return text grouped by page | Chunk or embed text |
| Chunker | Produce deterministic, overlapping chunks | Parse PDF bytes or call OpenAI |

### Why interfaces matter

The services will depend on small interfaces such as `EmbeddingProvider`,
`VectorStore`, and `AnswerGenerator`. OpenAI and Qdrant are adapters behind
those interfaces. This allows unit tests to use fakes and later permits a local
embedding model, another vector database, or another LLM without rewriting the
business workflow.

Interfaces will be defined by the package that consumes them, not collected in
one large generic package. This follows normal Go design and keeps each
contract small.

## Proposed folder structure

This is the target structure. Directories will be introduced only when their
phase needs them.

```text
rag-document-assistant/
├── main.go                         # composition root and process lifecycle
├── config/
│   ├── config.go                    # YAML loading, env overrides, validation
│   └── development.yml             # non-secret development defaults
├── controllers/
│   ├── health.go                    # health endpoint
│   ├── documents.go                 # upload/list HTTP handling
│   └── query.go                     # query HTTP handling
├── services/
│   ├── contracts.go                 # small provider/repository interfaces
│   ├── documents.go                 # ingestion use case
│   └── query.go                     # retrieval/generation use case
├── models/                          # Document, Chunk, and API/domain types
├── db/                              # PostgreSQL connection and repositories
├── router/
│   └── router.go                    # Gin route registration
├── server/
│   └── server.go                    # HTTP server start/stop
├── middleware/                      # request ID, recovery, structured logging
├── document/
│   └── pdf/                         # page-aware PDF text extraction
├── chunking/                       # simple character/rune chunker
├── embedding/
│   └── openai/                      # OpenAI embedding adapter
├── generation/
│   └── openai/                      # grounded answer adapter and prompt
├── vectorstore/
│   └── qdrant/                     # Qdrant implementation
├── migrations/                     # explicit PostgreSQL schema migrations
├── samples/                        # small synthetic PDFs/data added in Phase 6
├── .env.example
├── docker-compose.yml
├── Dockerfile
├── go.mod
└── README.md
```

The top-level layout intentionally follows the reference project. The main
difference is dependency ownership: instead of services reaching into a global
`db.DB` or global OpenAI helper, `main.go` constructs concrete clients once and
injects them into services, then injects services into controllers. This makes
startup, shutdown, and tests clearer while retaining the familiar structure.

## Data ownership

### PostgreSQL: durable application data

Proposed `documents` fields:

- `id` (UUID)
- `filename`
- `content_type`
- `size_bytes`
- `page_count`
- `status` (`processing`, `ready`, or `failed`)
- `error_message` (nullable, safe summary only)
- timestamps

Proposed `document_chunks` fields:

- `id` (UUID; also used as the Qdrant point ID)
- `document_id`
- `page_number` (one-based)
- `chunk_index` (zero-based within the document)
- `text`
- timestamps

PostgreSQL is the system of record. Keeping chunk text there as well as in the
Qdrant payload is deliberate duplication: Qdrant can be rebuilt, a failed
indexing run can be retried, and the source data remains inspectable with SQL.

### Qdrant: derived semantic-search index

One collection will contain document chunks. Each point will look conceptually
like:

```json
{
  "id": "same UUID as document_chunks.id",
  "vector": [0.012, -0.034, 0.056],
  "payload": {
    "document_id": "document UUID",
    "filename": "patient_anant.pdf",
    "page": 1,
    "chunk_index": 0,
    "text": "Mr Anant has jaundice."
  }
}
```

The collection's vector dimension must exactly match the chosen embedding
model. The application will validate this at startup rather than silently use
an incompatible collection.

## Chunking design

The first implementation will be page-aware and character-based. Each page is
chunked independently so a citation always maps to an unambiguous page.

- **Chunk size** is the maximum number of characters (more precisely, Unicode
  code points/runes) placed in one chunk.
- **Overlap** repeats the tail of one chunk at the beginning of the next.
- Overlap is necessary because an important sentence or idea may cross an
  arbitrary chunk boundary. Without overlap, neither chunk may contain enough
  meaning to match the question well.
- Too-small chunks lose context and create many embedding calls. Too-large
  chunks mix unrelated topics and consume more prompt space. Large overlap also
  creates redundant results and costs more to embed.

The initial values will be configurable; a reasonable starting experiment is
`CHUNK_SIZE=1200` and `CHUNK_OVERLAP=200`. These are not universal best values
and will be tested against the sample documents.

Character-based chunking is transparent and does not require a tokenizer, but
characters do not map consistently to model tokens. Token-based chunking gives
better control over embedding and LLM limits, especially across languages, but
couples the chunker to a tokenizer/model. We start character-based to expose the
algorithm, then can replace it through the chunker boundary.

## Retrieval and generation behavior

Retrieval initially uses cosine similarity only:

1. Embed the question using the same model used for document chunks.
2. Ask Qdrant for `RETRIEVAL_TOP_K` nearest points.
3. Remove results below `RETRIEVAL_MIN_SCORE`.
4. In Phase 4, return those matches directly so retrieval quality can be tested
   independently of the LLM.
5. In Phase 5, format the accepted chunks as labeled context and send them to
   the answer model with the question.

The Phase 5 system instruction will state, in substance:

```text
Use only the supplied context. Do not use outside knowledge and do not invent
facts. If the context does not contain enough evidence to answer the question,
reply exactly: I don't have enough information to answer that.
```

The minimum score is a useful first guardrail, not proof that a passage answers
the question. The LLM must still judge whether the retrieved text contains the
needed evidence. Tests will cover the exact fallback response.

## API plan

### `GET /health`

Phase 1 liveness endpoint. A later readiness check may report PostgreSQL and
Qdrant connectivity separately.

### `POST /api/documents`

Accepts one multipart field named `file`. Initially only text-based PDFs are
supported; scanned/image-only PDFs require OCR and are intentionally out of
scope.

The first implementation processes uploads synchronously because it is easier
to trace while learning. A document status still records partial failure.
Background jobs can be added later if large files make request times
unacceptable.

### `GET /api/documents`

Lists document metadata and ingestion status from PostgreSQL. It does not fetch
all chunk text or vectors.

### `POST /api/query` during Phase 4

Returns retrieval results without calling an answer model:

```json
{
  "question": "What disease does Anant have?",
  "matches": [
    {
      "chunk_id": "uuid",
      "document": "patient_anant.pdf",
      "page": 1,
      "chunk_index": 0,
      "text": "Mr Anant has jaundice.",
      "score": 0.91
    }
  ]
}
```

### `POST /api/query` from Phase 5 onward

```json
{
  "question": "What disease does Anant have?",
  "debug": false
}
```

```json
{
  "answer": "Anant has jaundice.",
  "sources": [
    {
      "document": "patient_anant.pdf",
      "page": 1,
      "chunk_id": "uuid",
      "score": 0.91
    }
  ]
}
```

When `debug` is true, the response will add a `debug` object containing the
pipeline stages, retrieved chunk text/scores, and the final context sent to the
LLM. It should report embedding model/dimension and a short vector preview,
not thousands of raw float values by default; returning the complete vector
would make the endpoint noisy and could expose more internal data without
improving the demonstration.

## Configuration plan

`config/development.yml` will provide safe, non-secret development defaults in
the same style as the reference project. Environment variables, documented in
`.env.example`, will override YAML values. Deployments can therefore change
configuration without rebuilding the application, and secrets remain outside
tracked files. Expected environment settings include:

```text
APP_ENV
HTTP_PORT
LOG_LEVEL
DATABASE_URL
QDRANT_URL
QDRANT_API_KEY
QDRANT_COLLECTION
OPENAI_API_KEY
OPENAI_EMBEDDING_MODEL
OPENAI_EMBEDDING_DIMENSIONS
OPENAI_CHAT_MODEL
CHUNK_SIZE
CHUNK_OVERLAP
RETRIEVAL_TOP_K
RETRIEVAL_MIN_SCORE
MAX_UPLOAD_BYTES
```

Secrets must never be placed in `development.yml` or committed `.env` files.
Structured JSON logging will use Go's `log/slog`, with request IDs and safe
fields. PDF text, prompts, API keys, and vectors will not be logged by default.

## Important tradeoffs to confirm before implementation

1. **PostgreSQL duplicates chunk text stored in Qdrant.** This costs some space
   but makes PostgreSQL the inspectable source of truth and Qdrant rebuildable.
   Proposed choice: accept the duplication.
2. **Synchronous ingestion.** It is simple to debug, but upload latency includes
   extraction, embedding, and indexing. Proposed choice: synchronous for the
   learning version; record status so asynchronous processing can be added
   later.
3. **Original PDF retention.** The RAG path only requires extracted text.
   Proposed choice: do not permanently store uploaded PDF bytes initially;
   retain document metadata and chunks. Reprocessing from the original would
   require re-uploading it.
4. **Character/rune chunks instead of token chunks.** This is easier to explain
   but less exact around model limits. Proposed choice: use rune-based windows
   in Phase 2 and keep the chunker replaceable.
5. **One chunk belongs to one page.** This gives reliable page citations but can
   lose continuity across page boundaries. Proposed choice: do not make chunks
   span pages initially.
6. **Explicit SQL and migrations instead of GORM `AutoMigrate`.** There is a
   little more code, but database behavior is visible and predictable.
   Proposed choice: use `pgx` plus versioned SQL migrations.
7. **Qdrant is not in the PostgreSQL transaction.** No distributed transaction
   exists between them. Proposed handling: stable UUIDs, idempotent upserts,
   document statuses, and cleanup/retry on partial failures.
8. **PDF extraction scope.** Pure text extraction does not handle scanned PDFs,
   tables, or complex layouts perfectly. Proposed choice: support ordinary
   text PDFs first and return a clear error when no text is extracted; OCR is a
   later feature.
9. **Familiar top-level package layout.** Controllers, services, models, router,
   server, database, and configuration remain at the repository root to match
   `../ai-health-bot`. Dependency injection and narrow interfaces are retained
   so the familiar layout does not introduce global vendor dependencies.

## Incremental implementation checklist

### Planning/bootstrap

- [x] Inspect the current directory
- [x] Review the architecture of sibling project `../ai-health-bot`
- [x] Propose folder structure and component boundaries
- [x] Explain ingestion, retrieval, and generation flows
- [x] Record important tradeoffs before coding
- [x] Create the phased checklist

### Phase 1 — runnable infrastructure and API shell

- [ ] Initialize the Go module and application composition root
- [ ] Add `config/development.yml`, environment overrides, validation, and
  `.env.example`
- [ ] Add Docker Compose services for PostgreSQL and Qdrant with health checks
- [ ] Add PostgreSQL connection and first explicit migration
- [ ] Add Gin server, request ID/recovery middleware, and structured logging
- [ ] Add `GET /health`
- [ ] Add graceful shutdown
- [ ] Document and run Phase 1 verification commands
- [ ] Stop for review before Phase 2

### Phase 2 — PDF extraction and chunking

- [ ] Add document/chunk domain models and PostgreSQL repositories
- [ ] Add `POST /api/documents` multipart validation and upload limits
- [ ] Extract text page by page from text-based PDFs
- [ ] Implement deterministic rune-based chunking with configurable overlap
- [ ] Persist document and chunk metadata with processing status
- [ ] Add `GET /api/documents`
- [ ] Add extraction and chunking unit tests
- [ ] Document and run Phase 2 verification commands
- [ ] Stop for review before Phase 3

### Phase 3 — embeddings and Qdrant indexing

- [ ] Define the embedding-provider and vector-store interfaces
- [ ] Add the OpenAI embedding adapter with batching
- [ ] Add the Qdrant adapter
- [ ] Create/validate the document-chunks collection and cosine metric
- [ ] Embed chunks and upsert vectors with required payload metadata
- [ ] Make indexing retry-safe and surface partial failures in document status
- [ ] Add adapter/service tests with fakes where appropriate
- [ ] Document and run Phase 3 verification commands
- [ ] Stop for review before Phase 4

### Phase 4 — retrieval without generation

- [ ] Embed incoming questions
- [ ] Search Qdrant with configurable Top-K
- [ ] Apply configurable minimum similarity threshold
- [ ] Implement the retrieval-only `POST /api/query` response
- [ ] Verify Anant/Vib-style semantic retrieval independently of an LLM
- [ ] Add retrieval service tests
- [ ] Document and run Phase 4 verification commands
- [ ] Stop for review before Phase 5

### Phase 5 — grounded generation and citations

- [ ] Define the answer-generator interface
- [ ] Add the OpenAI answer adapter
- [ ] Build the context format and strict grounded system prompt
- [ ] Return the exact insufficient-information fallback when unsupported
- [ ] Return filename/page/chunk/score citations
- [ ] Ensure citations are limited to context used for the answer
- [ ] Add grounded-answer and fallback tests
- [ ] Document and run Phase 5 verification commands
- [ ] Stop for review before Phase 6

### Phase 6 — observability, samples, and cleanup

- [ ] Add optional query debug output showing each RAG stage
- [ ] Add sample synthetic PDFs/data (no sensitive health information)
- [ ] Add integration tests for PostgreSQL and Qdrant
- [ ] Add final README architecture diagram and troubleshooting guide
- [ ] Review error responses, logs, timeouts, and resource cleanup
- [ ] Run formatting, static analysis, unit tests, and end-to-end demo
- [ ] Document final demonstration script
- [ ] Stop for final review

## What is intentionally out of scope

- Authentication and multi-tenancy
- Kubernetes, Kafka, and microservices
- OCR for scanned PDFs
- hybrid keyword/vector search, reranking, and agent/tool frameworks
- conversational memory
- a production-grade asynchronous job system
- LangChain or LlamaIndex

These can be useful later, but adding them now would hide the fundamental RAG
pipeline this project is meant to teach.
# rag-system

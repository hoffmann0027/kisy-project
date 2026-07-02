# KISY Enterprise Messenger - Enterprise Technical Specification (Part 1)

## 1. Vision

KISY is a closed corporate messenger intended exclusively for trusted
members of an organization.\
The system must prioritize security, stability, maintainability,
scalability and premium user experience.\
The generated solution must be production-ready and must not contain
placeholders or incomplete features.

## 2. Functional Goals

Implement authentication, authorization, invitation-based registration,
private messaging,\
group messaging, real-time communication through WebSockets, file
transfers, notifications,\
administration panel, audit logging, search, favorites, pinned chats,
avatars, user profiles,\
responsive design and cloud deployment.

## 3. Access Control

Registration is impossible without a CEO invitation token.\
Invitation tokens are generated only by Level 1.\
Tokens expire after exactly two minutes and become invalid after first
successful use.\
Users cannot delete accounts.\
Passwords can be changed.\
Username may be changed.\
CEO can reset credentials.

## 4. Security Model

Implement defense in depth.\
TLS 1.3 only.\
HTTPS everywhere.\
Argon2id password hashing.\
JWT Access + Refresh.\
Secure HTTPOnly cookies.\
Strict CSP.\
HSTS.\
Protection against XSS, CSRF, SQL injection, SSRF, clickjacking.\
Audit every privileged operation.\
Detect brute-force attacks.\
Hide internal errors from clients.\
Encrypt sensitive fields.\
Automatic backups.\
Structured logging.\
Malware scanning for uploads.\
Rate limiting.\
Account lockout after repeated failures.

## 5. Role Hierarchy

Levels 1-10.\
Level 1 (CEO) has unrestricted permissions.\
Each group defines minimum role.\
Higher groups are invisible to lower roles.\
Users cannot infer existence of inaccessible resources.\
Higher levels may initiate conversations with lower levels.\
Lower levels may reply only after conversation is opened or communicate
with same/lower roles.

## 6. Technology Stack

Frontend: React + TypeScript + Vite.\
Backend: Go.\
Database: PostgreSQL.\
Cache: Redis.\
Realtime: WebSockets.\
Proxy: Nginx.\
Deployment: Docker Compose.\
Documentation: OpenAPI.\
Testing: Unit + Integration + End-to-End.

## 7. UI

Telegram-inspired layout.\
Apple-inspired visual identity.\
Dark mode.\
Glassmorphism.\
Smooth animations.\
Responsive.\
Keyboard shortcuts.\
Accessibility.\
Premium typography.\
Minimalistic design.

## 8. AI Generation Requirements

Generate the ENTIRE project:\
Frontend.\
Backend.\
Database migrations.\
API.\
WebSockets.\
Docker.\
CI/CD.\
Testing.\
Documentation.\
No TODOs.\
No mock implementations.\
Enterprise quality architecture.

## Detailed Requirement Block 1

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 2

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 3

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 4

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 5

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 6

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 7

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 8

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 9

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 10

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 11

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 12

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 13

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 14

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 15

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 16

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 17

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 18

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 19

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

## Detailed Requirement Block 20

Describe in exhaustive detail the architecture, data flow, validation
rules, error handling, security constraints, UI behavior, API contracts,
logging, monitoring, performance targets, database indexing strategy,
caching policy, concurrency model, coding standards, testing strategy,
deployment workflow, disaster recovery, backup verification,
observability and future extensibility for this subsystem. The AI must
fully implement every requirement.

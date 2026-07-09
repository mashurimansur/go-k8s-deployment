# go-k8s-deployment — Go + Docker + Helm + GitHub Actions + K8s (home server)

## Alur pipeline

MR merge ke `master` → GitHub Actions jalan otomatis:
1. **test** — `go vet` + `go test` (runner cloud GitHub)
2. **build-and-push** — build image Docker, push ke `ghcr.io` (runner cloud GitHub)
3. **deploy** — `helm upgrade --install` ke cluster K8s kamu (runner **self-hosted**, jalan di home server)

Job `deploy` sengaja dipisah supaya jalan di self-hosted runner. Runner cloud GitHub
tidak akan pernah bisa menjangkau cluster K8s di rumah kamu tanpa expose API server
ke internet (tidak disarankan). Self-hosted runner membalik arahnya: runner di rumah
kamu yang "menghubungi keluar" ke GitHub untuk ambil job, bukan sebaliknya.

## Setup langkah demi langkah

### 1. Push project ini ke GitHub
Ganti semua `USERNAME` di `go.mod`, `values.yaml` dengan username/org GitHub kamu (sudah diset ke `mashurimansur`).

### 2. Pasang self-hosted runner di home server
Di repo GitHub → Settings → Actions → Runners → New self-hosted runner.
Ikuti instruksi yang muncul (download binary, `./config.sh`, `./run.sh`), lalu
tambahkan label `home-k8s` saat konfigurasi (atau tambahkan lewat UI runner).

Pastikan mesin runner ini punya:
- `kubectl` + `helm` terinstall
- `~/.kube/config` yang sudah mengarah ke cluster kamu (context aktif)

Jalankan sebagai service (systemd) supaya selalu aktif:
```bash
sudo ./svc.sh install
sudo ./svc.sh start
```

### 3. Buat image pull secret di cluster
Package di ghcr.io defaultnya **private**, jadi cluster butuh kredensial untuk pull.
Buat GitHub Personal Access Token (classic) dengan scope `read:packages`, lalu:

```bash
kubectl create namespace go-k8s-deployment
kubectl create secret docker-registry ghcr-secret \
  --namespace go-k8s-deployment \
  --docker-server=ghcr.io \
  --docker-username=mashurimansur \
  --docker-password=<PAT_read:packages> \
  --docker-email=you@example.com
```

Alternatif: buat package-nya public (Package settings di GitHub → Change visibility),
supaya tidak perlu imagePullSecret sama sekali — lebih simpel untuk belajar.

### 4. Push ke master
Sekali di-push / MR di-merge ke `master`, pipeline otomatis jalan dan men-deploy.

### 5. Cek hasil deploy
```bash
kubectl -n go-k8s-deployment get pods
kubectl -n go-k8s-deployment port-forward svc/go-k8s-deployment 8080:80
curl localhost:8080/healthz
```

## Development lokal
```bash
go run .
# atau
docker build -t go-k8s-deployment:local .
docker run -p 8080:8080 go-k8s-deployment:local
```

## Struktur project
```
.
├── main.go / main_test.go   # Go app
├── Dockerfile                # multi-stage build
├── .github/workflows/        # GitHub Actions pipeline
└── helm/go-k8s-deployment/    # Helm chart untuk deploy ke K8s
```

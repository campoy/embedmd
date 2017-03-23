COMMIT=$(git log -n1 --pretty='%h')
echo "Building embedmd with commit $COMMIT"
if TAG=$(git describe --exact-match --tags $COMMIT); then
    echo "Building tag $TAG"
else
    echo "You need to checkout a tag to be able to build"
    exit 1
fi

# TODO: check that the tag looks like vX.Y.Z

GOOSS=("darwin" "windows" "linux")
GOARCHS=("amd64" "386")

mkdir -p downloads/$TAG

for goos in "${GOOSS[@]}"; do
    for goarch in "${GOARCHS[@]}"; do
        echo "\n\nbuilding $goos $goarch"
        mkdir embedmd
        pushd embedmd
        GOOS=$goos GOARCH=$goarch go build -ldflags "-X main.version=$TAG" github.com/campoy/embedmd
        popd
        tar -cvzf "downloads/$TAG/embedmd.$TAG.$goos.$goarch.tar.gz" embedmd
        rm -rf embedmd
    done
done

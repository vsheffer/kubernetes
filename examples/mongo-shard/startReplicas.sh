#/bin/bash
host_port=37017
for rs in "rsa" "rsb" "rsc" ; do
    for hostnum in "ha" "hb" "hc" ; do
        sed -e "s/{REPLICA_SET_ID}/$rs/g" \
            -e "s/{HOSTNUM}/$hostnum/g" \
            -e "s/{HOST_PORT}/$host_port/g" \
            "${KUBE_ROOT}/examples/mongo-shard/mongo-replica.yaml" \
        | "${KUBE_ROOT}/cluster/kubectl.sh" create -f -

        sed -e "s/{REPLICA_SET_ID}/$rs/g" \
            -e "s/{HOSTNUM}/$hostnum/g" \
            "${KUBE_ROOT}/examples/mongo-shard/mongo-replica-service.yaml" | "${KUBE_ROOT}/cluster/kubectl.sh" create -f -
        host_port=$((host_port + 1))
    done
done

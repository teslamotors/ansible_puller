load("@io_bazel_rules_go//go:def.bzl", "go_binary", "go_library", "go_test")
load("@bazel_gazelle//:def.bzl", "gazelle")
load("@rules_pkg//:rpm.bzl", "pkg_rpm")
load("@rules_pkg//:pkg.bzl", "pkg_deb", "pkg_tar")
load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")
load("@rules_pkg//:mappings.bzl", "pkg_attributes", "pkg_filegroup", "pkg_files", "pkg_mkdirs")

# gazelle:prefix github.com/teslamotors/ansible_puller

gazelle(name = "gazelle")

buildifier(name = "buildifier")

#
# Build
#

go_library(
    name = "ansible_puller_lib",
    srcs = [
        "ansible.go",
        "http.go",
        "http_downloader.go",
        "idempotent_download.go",
        "main.go",
        "s3_downloader.go",
        "unarchive.go",
        "util.go",
        "venv.go",
    ],
    embedsrcs = glob([
        "templates/*.html",
    ]),
    importpath = "github.com/teslamotors/ansible_puller",
    visibility = ["//visibility:private"],
    deps = [
        "@com_github_aws_aws_sdk_go_v2//aws",
        "@com_github_aws_aws_sdk_go_v2_config//:config",
        "@com_github_aws_aws_sdk_go_v2_feature_s3_manager//:manager",
        "@com_github_aws_aws_sdk_go_v2_service_s3//:s3",
        "@com_github_gorilla_mux//:mux",
        "@com_github_pkg_errors//:errors",
        "@com_github_prometheus_client_golang//prometheus",
        "@com_github_prometheus_client_golang//prometheus/promhttp",
        "@com_github_satori_go_uuid//:go_uuid",
        "@com_github_sirupsen_logrus//:logrus",
        "@com_github_spf13_pflag//:pflag",
        "@com_github_spf13_viper//:viper",
    ],
)

#
# Tests
#

go_test(
    name = "ansible_puller_test",
    srcs = [
        "ansible_test.go",
        "http_downloader_test.go",
        "http_test.go",
        "s3_downloader_test.go",
        "unarchive_test.go",
    ],
    data = [
        ":ansible-puller.json",
        ":testdata",
    ],
    embed = [":ansible_puller_lib"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//suite",
    ],
)

go_test(
    name = "http_downloader_test",
    srcs = ["http_downloader_test.go"],
    data = [
        ":ansible-puller.json",
        ":testdata",
    ],
    embed = [":ansible_puller_lib"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//suite",
    ],
)

go_test(
    name = "http_test",
    srcs = ["http_test.go"],
    data = [
        ":ansible-puller.json",
        ":testdata",
    ],
    embed = [":ansible_puller_lib"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//suite",
    ],
)

go_test(
    name = "s3_downloader_test",
    srcs = ["s3_downloader_test.go"],
    data = [
        ":ansible-puller.json",
        ":testdata",
    ],
    embed = [":ansible_puller_lib"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//suite",
    ],
)

go_test(
    name = "unarchive_test",
    srcs = ["unarchive_test.go"],
    data = [
        ":ansible-puller.json",
        ":testdata",
    ],
    embed = [":ansible_puller_lib"],
    deps = [
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//suite",
    ],
)

#
# Packaging Constants
#

VERSION = "1"

RELEASE = "0"

DESCRIPTION = "This daemon extends the ansible-pull method of running Ansible. It uses S3 or HTTP file transmission instead of Git to manage distribution (easy to cache), and integrates with Prometheus monitoring."

#
# deb
#

go_binary(
    name = "ansible_puller_bin",
    basename = "ansible-puller",
    embed = [":ansible_puller_lib"],
    visibility = ["//visibility:public"],
)

pkg_tar(
    name = "pkg_tar_ansible_puller_bin",
    srcs = [":ansible_puller_bin"],
    mode = "0755",
    package_dir = "/opt/ansible-puller/",
)

genrule(
    name = "ansible_puller_example",
    srcs = [":ansible-puller.json"],
    outs = ["ansible-puller.json.example"],
    cmd = "cp $(SRCS) $(OUTS)",
)

pkg_tar(
    name = "pkg_tar_ansible_puller_config",
    srcs = [":ansible_puller_example"],
    mode = "0755",
    package_dir = "/etc/ansible-puller/",
)

pkg_tar(
    name = "pkg_tar_ansible_puller_systemd",
    srcs = [":ansible-puller.service"],
    mode = "0755",
    package_dir = "/etc/systemd/system/",
)

pkg_tar(
    name = "pkg_tar_ansible_puller",
    extension = "tar.gz",
    deps = [
        ":pkg_tar_ansible_puller_bin",
        ":pkg_tar_ansible_puller_config",
        ":pkg_tar_ansible_puller_systemd",
    ],
)

pkg_deb(
    name = "ansible_puller_deb",
    architecture = "amd64",
    data = ":pkg_tar_ansible_puller",
    description = DESCRIPTION,
    homepage = "https://github.com/teslamotors/ansible_puller",
    maintainer = "authors of ansible_puller",
    package = "ansible-puller",
    package_file_name = "ansible-puller.deb",
    version = VERSION,
)

#
# rpm
#

pkg_files(
    name = "pkg_file_ansible_puller_json",
    srcs = [":ansible-puller.json"],
    prefix = "/etc/ansible-puller/",
    renames = {":ansible-puller.json": "ansible-puller.json.example"},
)

pkg_files(
    name = "pkg_file_ansible_puller_bin",
    srcs = ["//:ansible_puller_bin"],
    attributes = pkg_attributes(mode = "0755"),
    prefix = "/opt/ansible-puller/",
)

pkg_files(
    name = "pkg_file_ansible_puller_service",
    srcs = [":ansible-puller.service"],
    prefix = "/etc/systemd/system/",
)

pkg_mkdirs(
    name = "pkg_mkdirs_ansible_puller_dirs",
    dirs = [
        "/var/log/ansible-puller/",
    ],
)

pkg_filegroup(
    name = "ansible_puller_rpm_files",
    srcs = [
        ":pkg_file_ansible_puller_bin",
        ":pkg_file_ansible_puller_json",
        ":pkg_file_ansible_puller_service",
        ":pkg_mkdirs_ansible_puller_dirs",
    ],
)

pkg_rpm(
    name = "ansible_puller_rpm",
    srcs = [":ansible_puller_rpm_files"],
    description = DESCRIPTION,
    license = "MIT",
    package_file_name = "ansible-puller.rpm",
    release = RELEASE,
    summary = "This daemon runs ansible in pull mode",
    version = VERSION,
)

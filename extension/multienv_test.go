// Copyright (C) 2017 NTT Innovation Institute, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extension_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudwan/gohan/extension"
	"github.com/cloudwan/gohan/extension/mocks"
	"github.com/golang/mock/gomock"
)

var _ = Describe("MultiEnvironment", func() {
	var (
		mockCtrl *gomock.Controller
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
	})

	Context("Cloning", func() {
		It("Should be lazy", func() {
			env := mock_extension.NewMockEnvironment(mockCtrl)
			multiEnv := extension.NewEnvironment([]extension.Environment{env})
			multiEnv.Clone()
			// no asserts, mock will complain if child env was cloned
		})

		It("Should happen on HandleEvent", func() {
			env := mock_extension.NewMockEnvironment(mockCtrl)
			env.EXPECT().Clone().Return(env)
			env.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil)
			env.EXPECT().IsEventHandled(gomock.Any(), gomock.Any()).Return(true)
			multiEnv := extension.NewEnvironment([]extension.Environment{env})
			multiEnv.Clone()

			Expect(multiEnv.HandleEvent("dummyEvent", map[string]interface{}{})).To(Succeed())
		})
		It("Should not happen when event is not handled", func() {
			env := mock_extension.NewMockEnvironment(mockCtrl)
			env.EXPECT().IsEventHandled(gomock.Any(), gomock.Any()).Return(false)
			multiEnv := extension.NewEnvironment([]extension.Environment{env})
			multiEnv.Clone()

			Expect(multiEnv.HandleEvent("dummyEvent", map[string]interface{}{})).To(Succeed())
		})
	})

	Context("Event handling", func() {
		It("Should not handle if no children", func() {
			multiEnv := extension.NewEnvironment([]extension.Environment{})
			Expect(multiEnv.IsEventHandled("dummyEvent", map[string]interface{}{})).To(BeFalse())
		})

		It("Should not handle if all children don't handle", func() {
			firstEnv := mock_extension.NewMockEnvironment(mockCtrl)
			firstEnv.EXPECT().IsEventHandled(gomock.Any(), gomock.Any()).Return(false)
			secondEnv := mock_extension.NewMockEnvironment(mockCtrl)
			secondEnv.EXPECT().IsEventHandled(gomock.Any(), gomock.Any()).Return(false)

			multiEnv := extension.NewEnvironment([]extension.Environment{firstEnv, secondEnv})
			Expect(multiEnv.IsEventHandled("dummyEvent", map[string]interface{}{})).To(BeFalse())
		})

		It("Should handle if at least one child handles", func() {
			firstEnv := mock_extension.NewMockEnvironment(mockCtrl)
			firstEnv.EXPECT().IsEventHandled(gomock.Any(), gomock.Any()).Return(false)
			secondEnv := mock_extension.NewMockEnvironment(mockCtrl)
			secondEnv.EXPECT().IsEventHandled(gomock.Any(), gomock.Any()).Return(true)

			multiEnv := extension.NewEnvironment([]extension.Environment{firstEnv, secondEnv})
			Expect(multiEnv.IsEventHandled("dummyEvent", map[string]interface{}{})).To(BeTrue())
		})
	})
})
